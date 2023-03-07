#include "K8SClient.h"
#include "K8SParams.h"
#include <boost/asio/posix/stream_descriptor.hpp>
#include <boost/asio/ssl/stream.hpp>
#include <boost/asio/ip/tcp.hpp>
#include <boost/asio/streambuf.hpp>
#include <boost/asio/write.hpp>
#include <boost/beast.hpp>
#include <boost/beast/ssl.hpp>

namespace beast = boost::beast;          // from <boost/beast.hpp>
namespace http = beast::http;            // from <boost/beast/http.hpp>
namespace asio = boost::asio;            // from <boost/asio.hpp>
namespace ssl = asio::ssl;               // from <boost/asio/ssl.hpp>
namespace json = boost::json;

constexpr size_t
MAX_K8S_CLIENT_SESSION_SIZE = 5;
static std::array<uint8_t, 8> NotifyBuffer{ "2323232" };

class K8SClientRequestEntry
{
public:
    std::string request_{};
    beast::flat_buffer buffer_{};
    beast::http::response <http::string_body> response_{};
    json::value document_{};
};

const char* K8SClientRequest::responseBody() const
{
    return entry_->response_.body().c_str();
}

size_t K8SClientRequest::responseSize() const
{
    return entry_->response_.body().size();
}

unsigned int K8SClientRequest::responseCode() const
{
    return entry_->response_.result_int();
}

const boost::json::value& K8SClientRequest::responseJson() const
{
    return entry_->document_;
}

class K8SClientWorker
{
public:
    K8SClientWorker(asio::io_context& ioContext, std::queue <std::shared_ptr<K8SClientRequest>>& pendingQueue,
            asio::posix::stream_descriptor& pipeReadStream,
            asio::posix::stream_descriptor& pipeWriteStream) : ioContext_(ioContext),
                                                               sslContext_(K8SParams::SSLContext()),
                                                               pendingTaskQueue_(pendingQueue),
                                                               pipeReadStream_(pipeReadStream),
                                                               pipeWriteStream_(pipeWriteStream)
    {
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
            pipeWriteStream_.async_write_some(notifyBufferWrapper_, [](const boost::system::error_code& ec, std::size_t)
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
            stream_ = std::make_shared < beast::ssl_stream < beast::tcp_stream >> (ioContext_, sslContext_);
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

        stream_->next_layer().async_connect(endpoint, [this](const boost::system::error_code& ec)
        { afterConnect(ec); });
    }

    void afterConnect(const boost::system::error_code& ec)
    {
        if (ec)
        {
            return afterSocketFail(std::string("connect to apiServer error: ").append(ec.message()));
        }
        doHandleShake();
    }

    void doHandleShake()
    {
        stream_->async_handshake(ssl::stream_base::client,
                [this](boost::system::error_code ec)
                {
                    afterHandshake(ec);
                });
    }

    void afterHandshake(boost::system::error_code ec)
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
        if (task == nullptr) //task may be canceled;
        {
            runningTask_.reset();
            doWaitNextTask();
            return;
        }
        task->setState(K8SClientRequestState::Running, "Writing RequestContent To ApiServer");
        auto&& entry = task->entry_;
        asio::async_write(*stream_, asio::buffer(entry->request_),
                [entry, this](boost::system::error_code ec, std::size_t transferred)
                {
                    afterWriteRequest(ec);
                });
    }

    void afterWriteRequest(const boost::system::error_code& ec)
    {
        if (ec)
        {
            return afterSocketFail(std::string("write to apiServer error: ").append(ec.message()));
        }
        doReadResponse();
    }

    void doReadResponse()
    {
        auto task = runningTask_.lock();
        if (task == nullptr)   //task may be canceled;
        {
            doWaitNextTask();
            return;
        }
        task->setState(K8SClientRequestState::Running, "Reading ResponseContent From ApiServer");
        auto&& entry = task->entry_;
        http::async_read(*stream_, entry->buffer_, task->entry_->response_,
                [this](boost::system::error_code ec, std::size_t transferred)
                {
                    afterReadResponse(ec, transferred);
                });
    }


    void afterReadResponse(const boost::system::error_code& ec, std::size_t transferred)
    {
        boost::ignore_unused(transferred);
        if (ec)
        {
            return afterSocketFail(std::string("read from apiServer error: ").append(ec.message()));
        }

        auto task = runningTask_.lock();
        if (task == nullptr) //task may be canceled;
        {
            doWaitNextTask();
            return;
        }

        boost::system::error_code er{};
        task->entry_->document_ = boost::json::parse(task->entry_->response_.body(), er);
        if (er)
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
        doWatchStream();
        doWatchNotify();
    }

    void doWatchNotify()
    {
        pipeReadStream_.async_read_some(notifyBufferWrapper_,
                [this](const boost::system::error_code& ec, std::size_t transferred)
                {
                    afterWatchNotify(ec);
                });
    }

    void afterWatchNotify(const boost::system::error_code& ec)
    {
        if (ec && ec != asio::error::operation_aborted)
        {
            throw std::runtime_error(
                    std::string("fatal error occurred while reading pipeReadStream :").append(ec.message()));
        }
        if (stream_ != nullptr)
        {
            stream_->next_layer().cancel();
        }
        doWork();
    }

    void doWatchStream()
    {
        if (stream_ != nullptr)
        {
            stream_->async_read_some(notifyBufferWrapper_,
                    [this](const boost::system::error_code& ec, std::size_t bytes_transferred)
                    {
                        afterWatchStream(ec, bytes_transferred);
                    });
        }
    }

    void afterWatchStream(const boost::system::error_code& ec, std::size_t)
    {
        if (ec && ec == asio::error::operation_aborted)
        {
            return;
        }
        stream_.reset();
    }

    void afterSocketFail(const std::string& message)
    {
        auto task = runningTask_.lock();
        if (task != nullptr)
        {
            task->setState(Error, message);
        }
        runningTask_.reset();
        stream_.reset();
        doWaitNextTask();
    }

private:
    asio::io_context& ioContext_;
    asio::ssl::context& sslContext_;
    std::queue <std::shared_ptr<K8SClientRequest>>& pendingTaskQueue_;
    asio::posix::stream_descriptor& pipeReadStream_;
    asio::posix::stream_descriptor& pipeWriteStream_;
    std::shared_ptr <beast::ssl_stream<beast::tcp_stream>> stream_{};
    std::weak_ptr <K8SClientRequest> runningTask_{};
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
        auto k8sClientSession = std::make_shared<K8SClientWorker>(ioContext_, pendingQueue_, pipeReadStream_,
                pipeWriteStream_);
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

std::shared_ptr <K8SClientRequest>
K8SClient::postRequest(K8SClientRequestMethod method, const std::string& url, const std::string& body)
{
    std::shared_ptr <K8SClientRequest> task(new K8SClientRequest());
    task->entry_ = std::make_shared<K8SClientRequestEntry>();
    switch (method)
    {
    case K8SClientRequestMethod::Patch:
    {
        task->entry_->request_ = buildPatchRequest(url, body);
    }
        break;
    case K8SClientRequestMethod::StrategicMergePatch:
    {
        task->entry_->request_ = buildSMPatchRequest(url, body);
    }
        break;
    case K8SClientRequestMethod::Post:
    {
        task->entry_->request_ = buildPostRequest(url, body);
    }
        break;
    case K8SClientRequestMethod::Delete:
    {
        task->entry_->request_ = buildDeleteRequest(url);
    }
        break;
    case K8SClientRequestMethod::Get:
    {
        task->entry_->request_ = buildGetRequest(url);
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
                        [](const boost::system::error_code ec, std::size_t transferred)
                        {
                            if (ec)
                            {
                                throw std::runtime_error(std::string(
                                        "fatal error occurred while writing pipeWriteStream: ").append(
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
