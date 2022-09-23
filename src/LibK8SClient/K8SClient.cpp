#include "K8SClient.h"
#include "K8SParams.h"
#include "HttpParser.h"
#include <asio/posix/stream_descriptor.hpp>
#include <asio/ssl/stream.hpp>
#include <asio/ip/tcp.hpp>
#include <asio/streambuf.hpp>
#include <asio/write.hpp>
#include <rapidjson/document.h>
#include <queue>
#include <mutex>
#include <iostream>
#include <atomic>
#include <thread>
#include <condition_variable>

constexpr size_t MAX_K8S_CLIENT_SESSION_SIZE = 5;
static std::array<uint8_t, 8> NotifyBuffer{ "2323232" };

class K8SClientRequestEntry
{
public:
	void initHttpResponseParser()
	{
		memset(&httpResponseParser_, 0, sizeof(httpResponseParser_));
		http_parser_init(&httpResponseParser_, HTTP_RESPONSE);
		httpResponseParser_.data = this;
	}

	std::string httpRequestContent_{};
	std::string httpResponseBody_{};
	http_parser httpResponseParser_{};
	bool httpResponseComplete_{ false };
	rapidjson::Document document_{};
};

const char* K8SClientRequest::responseBody() const
{
	return entry_->httpResponseBody_.data();
}

size_t K8SClientRequest::responseSize() const
{
	return entry_->httpResponseBody_.size();
}

unsigned int K8SClientRequest::responseCode() const
{
	return entry_->httpResponseParser_.status_code;
}

const rapidjson::Value& K8SClientRequest::responseJson() const
{
	return entry_->document_;
}

class K8SClientWorker
{
public:
	K8SClientWorker(asio::io_context& ioContext, std::queue<std::shared_ptr<K8SClientRequest>>& pendingQueue, asio::posix::stream_descriptor& pipeReadStream,
			asio::posix::stream_descriptor& pipeWriteStream)
			: ioContext_(ioContext), sslContext_(K8SParams::SSLContext()), pendingTaskQueue_(pendingQueue), pipeReadStream_(pipeReadStream), pipeWriteStream_(pipeWriteStream)
	{
		memset(&responseParserSetting_, 0, sizeof(responseParserSetting_));
		responseParserSetting_.on_body = ([](http_parser* p, const char* d, size_t l) -> int
		{
			auto* taskEntry = static_cast<K8SClientRequestEntry* const>(p->data);
			taskEntry->httpResponseBody_.append(d, l);
			return 0;
		});
		responseParserSetting_.on_message_complete = [](http_parser* p) -> int
		{
			auto* taskEntry = static_cast<K8SClientRequestEntry* const>(p->data);
			taskEntry->httpResponseComplete_ = true;
			return 0;
		};
	};

	~K8SClientWorker() = default;

	void run()
	{
		doWork();
	}

private:
	void doWork()
	{
		if (pendingTaskQueue_.empty())
		{
			return doWaitNextTask();
		}

		runningTask_ = pendingTaskQueue_.front();
		pendingTaskQueue_.pop();

		if (!pendingTaskQueue_.empty())
		{
			pipeWriteStream_.async_write_some(notifyBufferWrapper_,
					[](const std::error_code& ec, std::size_t)
					{
						if (ec)
						{
							throw std::runtime_error(std::string(
									"fatal error occurred while writing eventStream :").append(
									ec.message()));
						}
					});
		}

		if (stream_ == nullptr)
		{
			stream_.reset(new asio::ssl::stream<asio::ip::tcp::socket>(ioContext_, sslContext_));
			doConnect();
			return;
		}

		if (!stream_->next_layer().is_open())
		{
			doConnect();
			return;
		}

		doWriteRequest();
	}

	void doConnect()
	{
		const auto& apiServerIP = K8SParams::APIServerHost();
		if (apiServerIP.empty())
		{
			return afterSocketFail(std::string("empty host value"));
		}
		const auto& apiServerPort = K8SParams::APIServerPort();
		if (apiServerPort <= 0 || apiServerPort > 65535)
		{
			return afterSocketFail(std::string("bad port value|").append(std::to_string(apiServerPort)));
		}

		asio::ip::tcp::endpoint endpoint(asio::ip::address::from_string(apiServerIP), apiServerPort);
		stream_->next_layer().async_connect(endpoint, [this](asio::error_code ec)
		{ afterConnect(ec); });
	}

	void afterConnect(const asio::error_code& ec)
	{
		if (ec)
		{
			return afterSocketFail(std::string("connect to apiServer error: ").append(ec.message()));
		}
		doHandleShake();
	}

	void doHandleShake()
	{
		stream_->async_handshake(asio::ssl::stream_base::client,
				[this](asio::error_code ec)
				{
					afterHandshake(ec);
				});
	}

	void afterHandshake(asio::error_code ec)
	{
		if (ec)
		{
			return afterSocketFail(std::string("handshake with apiServer error: ").append(ec.message()));
		}
		doWriteRequest();
	}

	void doWriteRequest()
	{
		auto task = runningTask_.lock();
		if (task == nullptr)
		{ //task may be canceled;
			doWaitNextTask();
			return;
		}
		task->setState(K8SClientRequestState::Running, "Writing RequestContent To ApiServer");
		auto taskEntry = task->entry_;
		asio::async_write(*stream_, asio::buffer(taskEntry->httpRequestContent_),
				[taskEntry, this](asio::error_code ec, std::size_t bytes_transferred)
				{
					afterWriteRequest(ec);
				});
	}

	void afterWriteRequest(const asio::error_code& ec)
	{
		if (ec)
		{
			return afterSocketFail(std::string("write to apiServer error: ").append(ec.message()));
		}
		responseBuffer_.consume(responseBuffer_.size());
		doReadResponse();
	}

	void doReadResponse()
	{
		auto task = runningTask_.lock();
		if (task == nullptr)
		{  //task may be canceled;
			doWaitNextTask();
			return;
		}

		task->setState(K8SClientRequestState::Running, "Reading ResponseContent From ApiServer");
		stream_->async_read_some(responseBuffer_.prepare(1024 * 1024 * 4),
				[this](asio::error_code ec, size_t bytes_transferred)
				{
					afterReadResponse(ec, bytes_transferred);
				});
	}

	void afterReadResponse(const asio::error_code& ec, std::size_t bytes_transferred)
	{
		if (ec)
		{
			return afterSocketFail(std::string("read from apiServer error: ").append(ec.message()));
		}
		responseBuffer_.commit(bytes_transferred);

		auto task = runningTask_.lock();
		if (task == nullptr)
		{  //task may be canceled;
			doWaitNextTask();
			return;
		}

		const char* willParseData = asio::buffer_cast<const char*>(responseBuffer_.data());
		size_t parserLength = http_parser_execute(&task->entry_->httpResponseParser_, &responseParserSetting_,
				willParseData, responseBuffer_.size());
		responseBuffer_.consume(parserLength);

		if (!task->entry_->httpResponseComplete_)
		{
			doReadResponse();
			return;
		}

		task->entry_->document_.Parse(task->entry_->httpResponseBody_.data(), task->entry_->httpResponseBody_.size());
		if (task->entry_->document_.HasParseError())
		{
			task->setState(K8SClientRequestState::Error, "decode response to json error");
		}
		else
		{
			task->setState(K8SClientRequestState::Done, "success");
		}
		runningTask_.reset();
		doWaitNextTask();
	}

	void doWaitNextTask()
	{
		doWaitEvent();
		doWaitSocket();
	}

	void doWaitEvent()
	{
		pipeReadStream_.async_read_some(notifyBufferWrapper_,
				[this](const std::error_code& ec, std::size_t bytes_transferred)
				{
					afterWaitEvent(ec);
				});
	}

	void afterWaitEvent(const asio::error_code& ec)
	{
		if (ec && ec != asio::error::operation_aborted)
		{
			throw std::runtime_error(
					std::string("fatal error occurred while reading eventStream :").append(ec.message()));
		}
		if (stream_ != nullptr && stream_->next_layer().is_open())
		{
			asio::error_code tempErrorCode{};
			stream_->next_layer().cancel(tempErrorCode);
		}
		doWork();
	}

	void doWaitSocket()
	{
		if (stream_ == nullptr)
		{
			return;
		}

		if (!stream_->next_layer().is_open())
		{
			return;
		}

		stream_->async_read_some(notifyBufferWrapper_,
				[this](const std::error_code& ec, std::size_t bytes_transferred)
				{
					afterWaitSocket(ec, bytes_transferred);
				});
	}

	void afterWaitSocket(const asio::error_code& ec, std::size_t)
	{
		if (ec && ec == asio::error::operation_aborted)
		{
			return;
		}
		stream_.reset(new asio::ssl::stream<asio::ip::tcp::socket>(ioContext_, sslContext_));
	}

	void afterSocketFail(const std::string& message)
	{
		auto task = runningTask_.lock();
		if (task != nullptr)
		{
			task->setState(Error, message);
		}
		runningTask_.reset();
		stream_.reset(new asio::ssl::stream<asio::ip::tcp::socket>(ioContext_, sslContext_));
		doWaitNextTask();
	}

private:
	asio::io_context& ioContext_;
	asio::ssl::context& sslContext_;
	std::queue<std::shared_ptr<K8SClientRequest>>& pendingTaskQueue_;
	asio::posix::stream_descriptor& pipeReadStream_;
	asio::posix::stream_descriptor& pipeWriteStream_;
	std::unique_ptr<asio::ssl::stream<asio::ip::tcp::socket>> stream_{};
	std::weak_ptr<K8SClientRequest> runningTask_{};
	http_parser_settings responseParserSetting_{};
	asio::streambuf responseBuffer_{};
	asio::mutable_buffer notifyBufferWrapper_{ NotifyBuffer.begin(), NotifyBuffer.size() };
};

void K8SClient::start()
{
	int pipe[2] = {};

	int res = ::pipe(pipe);
	if (res != 0)
	{
		auto errorMessage = std::string("creat pipe failed: ") + ::strerror(errno);
		throw std::runtime_error(errorMessage);
	}

	res = ::fcntl(pipe[0], F_SETFL, O_NONBLOCK);
	if (res != 0)
	{
		auto errorMessage = std::string("fcntl pipe failed: ") + ::strerror(errno);
		throw std::runtime_error(errorMessage);
	}

	pipeReadStream_.assign(pipe[0]);
	pipeWriteStream_.assign(pipe[1]);

	for (size_t i = 0; i < MAX_K8S_CLIENT_SESSION_SIZE; ++i)
	{
		auto k8sClientSession = std::make_shared<K8SClientWorker>(ioContext_, pendingQueue_, pipeReadStream_, pipeWriteStream_);
		k8sClientSession->run();
		sessionVector_.push_back(k8sClientSession);
	}
	thread_ = std::thread([this]
	{ ioContext_.run(); });
	thread_.detach();
}

void K8SClient::stop()
{
	ioContext_.stop();
}

std::shared_ptr<K8SClientRequest> K8SClient::postRequest(K8SClientRequestMethod method, const std::string& url, const std::string& body)
{
	std::shared_ptr<K8SClientRequest> task(new K8SClientRequest());
	task->entry_ = std::make_shared<K8SClientRequestEntry>();
	task->entry_->initHttpResponseParser();
	switch (method)
	{
	case K8SClientRequestMethod::Patch:
	{
		task->entry_->httpRequestContent_ = buildPatchRequest(url, body);
	}
		break;
	case K8SClientRequestMethod::StrategicMergePatch:
	{
		task->entry_->httpRequestContent_ = buildSMPatchRequest(url, body);
	}
		break;
	case K8SClientRequestMethod::Post:
	{
		task->entry_->httpRequestContent_ = buildPostRequest(url, body);
	}
		break;
	case K8SClientRequestMethod::Delete:
	{
		task->entry_->httpRequestContent_ = buildDeleteRequest(url);
	}
		break;
	case K8SClientRequestMethod::Get:
	{
		task->entry_->httpRequestContent_ = buildGetRequest(url);
	}
		break;
	default:
		return nullptr;
	}
	ioContext_.post(
			[this, task]
			{
				pendingQueue_.push(task);
				pipeWriteStream_.async_write_some(asio::buffer(NotifyBuffer.begin(), NotifyBuffer.size()),
						[](const std::error_code ec, std::size_t bytes_transferred)
						{
							if (ec)
							{
								throw std::runtime_error(std::string(
										"fatal error occurred while writing  eventStream :").append(
										ec.message()));
							}
						});
			}
	);
	return task;
}

std::string K8SClient::buildPostRequest(const std::string& url, const std::string& body)
{
	std::ostringstream strStream;
	strStream << "POST " << url << " HTTP/1.1\r\n";
	strStream << "Authorization: Bearer " << K8SParams::ClientToken() << "\r\n";
	strStream << "Host: " << K8SParams::APIServerHost() << ":" << K8SParams::APIServerPort() << "\r\n";
	strStream << "Content-Length: " << body.size() << "\r\n";
	strStream << "Content-Type: application/json\r\n";
	strStream << "Connection: Keep-Alive\r\n";
	strStream << "\r\n";
	strStream << body;
	std::string requestContent = strStream.str();
	return requestContent;
}

std::string K8SClient::buildPatchRequest(const std::string& url, const std::string& body)
{
	std::ostringstream strStream;
	strStream << "PATCH " << url << " HTTP/1.1\r\n";
	strStream << "Authorization: Bearer " << K8SParams::ClientToken() << "\r\n";
	strStream << "Host: " << K8SParams::APIServerHost() << ":" << K8SParams::APIServerPort() << "\r\n";
	strStream << "Content-Length: " << body.size() << "\r\n";
	strStream << "Content-Type: application/json-patch+json\r\n";
	strStream << "Connection: Keep-Alive\r\n";
	strStream << "\r\n";
	strStream << body;
	std::string requestContent = strStream.str();
	return requestContent;
}

std::string K8SClient::buildDeleteRequest(const std::string& url)
{
	std::ostringstream strStream;
	strStream << "DELETE " << url << " HTTP/1.1\r\n";
	strStream << "Authorization: Bearer " << K8SParams::ClientToken() << "\r\n";
	strStream << "Host: " << K8SParams::APIServerHost() << ":" << K8SParams::APIServerPort() << "\r\n";
	strStream << "Connection: Keep-Alive\r\n";
	strStream << "\r\n";
	std::string requestContent = strStream.str();
	return requestContent;
}

std::string K8SClient::buildSMPatchRequest(const std::string& url, const std::string& body)
{
	std::ostringstream strStream;
	strStream << "PATCH " << url << " HTTP/1.1\r\n";
	strStream << "Authorization: Bearer " << K8SParams::ClientToken() << "\r\n";
	strStream << "Host: " << K8SParams::APIServerHost() << ":" << K8SParams::APIServerPort() << "\r\n";
	strStream << "Content-Length: " << body.size() << "\r\n";
	strStream << "Content-Type: application/strategic-merge-patch+json\r\n";
	strStream << "Connection: Keep-Alive\r\n";
	strStream << "\r\n";
	strStream << body;
	std::string requestContent = strStream.str();
	return requestContent;
}

std::string K8SClient::buildGetRequest(const std::string& url)
{
	std::ostringstream strStream;
	strStream << "GET " << url << " HTTP/1.1\r\n";
	strStream << "Authorization: Bearer " << K8SParams::ClientToken() << "\r\n";
	strStream << "Host: " << K8SParams::APIServerHost() << ":" << K8SParams::APIServerPort() << "\r\n";
	strStream << "Content-Type: application/json\r\n";
	strStream << "Connection: Keep-Alive\r\n";
	strStream << "\r\n";
	std::string requestContent = strStream.str();
	return requestContent;
}
