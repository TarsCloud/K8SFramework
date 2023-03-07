
#include "K8SWatcher.h"
#include "K8SWatcherError.h"
#include "K8SParams.h"
#include <iostream>
#include <boost/asio/ssl/stream.hpp>
#include <boost/asio/ip/tcp.hpp>
#include <boost/asio/posix/stream_descriptor.hpp>
#include <boost/asio/write.hpp>
#include <boost/beast.hpp>
#include <boost/beast/ssl.hpp>

namespace beast = boost::beast;          // from <boost/beast.hpp>
namespace http = beast::http;            // from <boost/beast/http.hpp>
namespace asio = boost::asio;            // from <boost/asio.hpp>
namespace ssl = asio::ssl;               // from <boost/asio/ssl.hpp>
namespace json = boost::json;


constexpr char ADDEDTypeValue[] = "ADDED";
constexpr char DELETETypeValue[] = "DELETED";
constexpr char UPDATETypeValue[] = "MODIFIED";
constexpr char BOOKMARKTypeValue[] = "BOOKMARK";
constexpr char ERRORTypeValue[] = "ERROR";

std::string K8SWatcherSetting::listUri() const
{
    std::ostringstream os;
    os << path_ << "?limit=" << limit_;
    if (!labelFilter_.empty())
    {
        os << "&labelSelector=" << labelFilter_;
    }
    if (!filedFilter_.empty())
    {
        os << "&filedSelector=" << filedFilter_;
    }
    if (!continue_.empty())
    {
        os << "&" << "continue=" << continue_;
    }
    return os.str();
}

std::string K8SWatcherSetting::watchUri() const
{
    std::ostringstream os;
    os << path_ << "?watch=true&allowWatchBookmarks=true&timeoutSeconds=" << (std::rand() % 19) * 60 + overtime_;
    if (!labelFilter_.empty())
    {
        os << "&labelSelector=" << labelFilter_;
    }
    if (!filedFilter_.empty())
    {
        os << "&filedSelector=" << filedFilter_;
    }
    os << "&resourceVersion=" << (newestVersion_.empty() ? "0" : newestVersion_);
    return os.str();
}

class K8SWatcherSession :public std::enable_shared_from_this<K8SWatcherSession>
{
public:
    explicit K8SWatcherSession(asio::io_context& ioContext, std::atomic<int>& waitSyncCount,
            std::condition_variable& conditionVariable,
            const K8SWatcherSetting& callback)
            :ioContext_(ioContext),
             waitSyncCount_(waitSyncCount),
             conditionVariable_(conditionVariable),
             stream_(ioContext_, K8SParams::SSLContext()),
             callback_(callback)
    {
    }

    void start()
    {
        doConnect();
    }

private:
    void afterError(const boost::system::error_code& ec, const std::string& message) const
    {
        waitSyncCount_--;
        conditionVariable_.notify_all();
        if (callback_.onError)
        {
            auto again = callback_.onError(ec, message);
            if (again)
            {
                K8SWatcher::instance().addWatch(callback_);
            }
            return;
        }
        std::cout << "K8SWatcher Error, Reason: " << ec.message() << ", Message: " << message << std::endl;
    }

    void doConnect()
    {
        const auto& apiServerIP = K8SParams::APIServerHost();
        if (apiServerIP.empty())
        {
            std::string msg = "empty apiServerHost value";
            return afterError(K8SWatcherError::BadParams, msg);
        }

        const auto& apiServerPort = K8SParams::APIServerPort();
        if (apiServerPort <= 0 || apiServerPort > 65535)
        {
            std::string msg = "bad apiServerPort value|" + std::to_string(K8SParams::APIServerPort());
            return afterError(K8SWatcherError::BadParams, msg);
        }

        asio::ip::tcp::endpoint endpoint(asio::ip::address::from_string(apiServerIP), apiServerPort);
        auto self = shared_from_this();
        stream_.next_layer().async_connect(endpoint, [self](const boost::system::error_code& ec)
        {self->afterConnect(ec);});
    }

    void afterConnect(const boost::system::error_code& ec)
    {
        if (ec)
        {
            return afterError(ec, "error when connect to apiServer");
        }
        doHandleShake();
    };

    void doHandleShake()
    {
        auto self = shared_from_this();
        stream_.async_handshake(ssl::stream_base::client,
                [self](const boost::system::error_code& ec)
                {
                    self->afterHandshake(ec);
                });
    }

    void afterHandshake(const boost::system::error_code& ec)
    {
        if (ec)
        {
            return afterError(ec, "error when handshake with apiServer");
        }
        if (callback_.preList)
        {
            callback_.preList();
        }
        doListRequest();
    }

    void doListRequest()
    {
        std::ostringstream strStream;
        strStream << "GET " << callback_.listUri() << " HTTP/1.1\r\n";
        strStream << "Host: " << K8SParams::APIServerHost() << ":" << K8SParams::APIServerPort() << "\r\n";
        strStream << "Authorization: Bearer " << K8SParams::ClientToken() << "\r\n";
        strStream << "Connection: Keep-Alive\r\n";
        strStream << "\r\n";
        requestContent_ = strStream.str();
        auto self = shared_from_this();
        asio::async_write(stream_, asio::buffer(requestContent_),
                [self](const boost::system::error_code& ec, size_t transferred)
                {self->afterListRequest(ec, transferred);});
    }

    void afterListRequest(const boost::system::error_code& ec, std::size_t transferred)
    {
        boost::ignore_unused(transferred);
        if (ec)
        {
            return afterError(ec, "error when write list request to apiServer");
        }
        buffer_.clear();
        responseParser_ = std::make_shared<http::response_parser < http::string_body>>
        ();
        doReadListResponse();
    }

    void doReadListResponse()
    {
        auto self = shared_from_this();
        http::async_read(stream_, buffer_, *responseParser_,
                [self](const boost::system::error_code& ec, size_t transferred)
                {
                    self->afterReadListResponse(ec, transferred);
                });
    }

    void afterReadListResponse(const boost::system::error_code& ec, size_t transferred)
    {
        boost::ignore_unused(transferred);
        if (ec)
        {
            return afterError(ec, "error when read list response from apiServer");
        }

        boost::system::error_code er{};
        assert(responseParser_->is_done());
        auto&& response = responseParser_->release();
        auto v = boost::json::parse(response.body(), er);
        if (er)
        {
            return afterError(K8SWatcherError::UnexpectedResponse, response.body());
        }

        auto&& pItems = v.find_pointer("/items", er);
        if (er || pItems == nullptr || !pItems->is_array())
        {
            return afterError(er, "error when read list response from apiServer");
        }

        if (callback_.onAdded)
        {
            for (auto&& item:pItems->get_array())
            {
                callback_.onAdded(item, K8SWatchEventDrive::List);
            }
        }
        auto pResourceVersion = v.find_pointer("/metadata/resourceVersion", er);
        if (er || pResourceVersion == nullptr || !pResourceVersion->is_string())
        {
            return afterError(er, "error when read list response from apiServer");
        }

        callback_.newestVersion_ = boost::json::value_to<std::string>(*pResourceVersion);

        auto pContinue = v.find_pointer("/metadata/continue", er);
        if (pContinue == nullptr)
        {
            callback_.continue_ = "";
        }
        else
        {
            assert(pContinue->is_string());
            callback_.continue_ = boost::json::value_to<std::string>(*pContinue);
        }
        if (callback_.continue_.empty())
        {
            if (callback_.postList)
            {
                callback_.postList();
            }
            waitSyncCount_--;
            conditionVariable_.notify_all();
            doWatchRequest();
            return;
        }
        doListRequest();
    }

    void doWatchRequest()
    {
        std::ostringstream strStream;
        strStream << "GET " << callback_.watchUri() << " HTTP/1.1\r\n";
        strStream << "Host: " << K8SParams::APIServerHost() << ":" << K8SParams::APIServerPort() << "\r\n";
        strStream << "Authorization: Bearer " << K8SParams::ClientToken() << "\r\n";
        strStream << "Connection: Keep-Alive\r\n";
        strStream << "\r\n";
        requestContent_ = strStream.str();
        auto self = shared_from_this();
        asio::async_write(stream_, asio::buffer(requestContent_),
                [self](boost::system::error_code ec, size_t transferred)
                {self->afterWatchRequest(ec, transferred);});
    }

    void afterWatchRequest(const boost::system::error_code& ec, std::size_t transferred)
    {
        boost::ignore_unused(transferred);
        if (ec)
        {
            return afterError(ec, "error when write watch request to apiServer");
        }
        buffer_.clear();
        responseParser_ = std::make_shared<http::response_parser < http::string_body>>
        ();
        doReadWatchResponse();
    }

    void doReadWatchResponse()
    {
        auto self = shared_from_this();
        http::async_read_some(stream_, buffer_, *responseParser_,
                [self](const boost::system::error_code& ec, size_t transferred)
                {
                    self->afterReadWatchResponse(ec, transferred);
                });
    }

    void afterReadWatchResponse(const boost::system::error_code& ec, size_t transferred)
    {
        boost::ignore_unused(transferred);
        if (ec)
        {
            return afterError(ec, "error when read watch response from apiServer");
        }
        if (!responseParser_->is_header_done())
        {
            doReadWatchResponse();
            return;
        }
        auto&& response = responseParser_->get();
        auto&& body = response.body();

        constexpr unsigned int HTTP_OK = 200;

        if (response.result_int() != HTTP_OK)
        {
            if (!responseParser_->is_done())
            {
                doReadWatchResponse();
                return;
            }
            return afterError(K8SWatcherError::UnexpectedResponse, body);
        }

        if (!body.empty())
        {
            boost::system::error_code err{};
            auto data = body.c_str();
            auto size = body.size();
            while (size > 0)
            {
                auto tell = jsonStreamParser_.write_some(data, size, err);
                if (err)
                {
                    return afterError(K8SWatcherError::UnexpectedResponse, body);
                }

                assert(size >= tell);
                size -= tell;
                data += tell;

                if (jsonStreamParser_.done())
                {
                    auto&& document = jsonStreamParser_.release();
                    jsonStreamParser_.reset();

                    auto pType = document.at_pointer("/type");
                    assert(pType != nullptr && pType.is_string());
                    auto type = pType.as_string().c_str();
                    if (strcmp(ERRORTypeValue, type) == 0)
                    {
                        auto pErrorCode = document.at_pointer("/object/code");
                        auto errorCode = pErrorCode.as_int64();
                        constexpr auto HTTP_GONE = 410;

                        if (errorCode != HTTP_GONE)
                        {
                            return afterError(K8SWatcherError::UnexpectedResponse, body);
                        }

                        if (responseParser_->is_done())
                        {
                            if (callback_.preList)
                            {
                                callback_.preList();
                            }
                            return doListRequest();
                        }
                        return doReadWatchResponse();
                    }
                    dispatchEvent(type, document);
                }
            }
            if (size == 0)
            {
                body.clear();
            }
            else
            {
                body.replace(0, size, data);
                body.resize(size);
            }
        }
        responseParser_->is_done() ? doWatchRequest() : doReadWatchResponse();
    }

    void dispatchEvent(const char* type, const boost::json::value& document)
    {
        boost::system::error_code er{};
        auto pResourceVersion = document.find_pointer("/object/metadata/resourceVersion", er);
        if (er || pResourceVersion == nullptr || !pResourceVersion->is_string())
        {
            return;
        }

        callback_.newestVersion_ = boost::json::value_to<std::string>(*pResourceVersion);
        if (::strcmp(BOOKMARKTypeValue, type) == 0)
        {
            return;
        }
        const auto& object = document.at("object");

        if (::strcmp(ADDEDTypeValue, type) == 0)
        {
            callback_.onAdded(object, K8SWatchEventDrive::Watch);
            return;
        }
        if (::strcmp(UPDATETypeValue, type) == 0)
        {
            callback_.onModified(object);
            return;
        }
        if (::strcmp(DELETETypeValue, type) == 0)
        {
            callback_.onDeleted(object);
            return;
        }
    }

private:
    asio::io_context& ioContext_;
    std::atomic<int>& waitSyncCount_;
    std::condition_variable& conditionVariable_;
    beast::ssl_stream<beast::tcp_stream> stream_;
    K8SWatcherSetting callback_;
    std::string requestContent_{};
    json::stream_parser jsonStreamParser_{};
    beast::flat_buffer buffer_{};
    std::shared_ptr<http::response_parser < http::string_body>> responseParser_;
};

K8SWatcherSetting::K8SWatcherSetting(const std::string& group, const std::string& version, const std::string& plural,
        const std::string& _namespace)
{
    std::ostringstream os;
    if (group.empty())
    {
        os << "/api/" << version;
    }
    else
    {
        os << "/apis/" << group << "/" << version;
    }
    assert(!version.empty());
    if (!_namespace.empty())
    {
        os << "/namespaces/" << _namespace;
    }
    assert(!plural.empty());
    os << "/" << plural;
    path_ = os.str();
}

void K8SWatcher::addWatch(const K8SWatcherSetting& setting)
{
    waitSyncCount_++;
    auto session = std::make_shared<K8SWatcherSession>(ioContext_, waitSyncCount_, conditionVariable_, setting);
    session->start();
}

void K8SWatcher::start()
{
    thread_ = std::thread([this]
    {
        asio::io_context::work work(ioContext_);
        ioContext_.run();
    });
    thread_.detach();
}

void K8SWatcher::stop()
{
    ioContext_.stop();
}
