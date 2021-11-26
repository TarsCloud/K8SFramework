
#include "K8SWatcher.h"
#include "K8SWatcherError.h"
#include "K8SParams.h"
#include "HttpParser.h"
#include <asio/ssl/stream.hpp>
#include <asio/streambuf.hpp>
#include <asio/ip/tcp.hpp>
#include <rapidjson/pointer.h>
#include <rapidjson/document.h>
#include <iostream>

enum class K8SWatchEvent
{
    K8SWatchEventAdded = 1u,
    K8SWatchEventDeleted = 2u,
    K8SWatchEventModified = 3u,
    K8SWatchEventBookmark = 4u,
    K8SWatchEventError = 5u,
};

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
    os << "&resourceVersion=" << newestVersion_;
    return os.str();
}

class K8SWatcherSession : public std::enable_shared_from_this<K8SWatcherSession>
{
    struct ResponseParserState
    {
        ResponseParserState() = default;

        bool headComplete_{ false };
        bool messageComplete_{ false };
        bool bodyArrive_{ false };
        unsigned int code_{ 0 };
        std::string bodyBuffer_{};

        void cleanup()
        {
            headComplete_ = false;
            bodyArrive_ = false;
            messageComplete_ = false;
            bodyBuffer_.clear();
            code_ = 0;
        }
    };

 public:
    explicit K8SWatcherSession(asio::io_context& ioContext, std::atomic<int>& waitSyncCount, std::condition_variable& conditionVariable,
        const K8SWatcherSetting& callback)
        :
        ioContext_(ioContext),
        waitSyncCount_(waitSyncCount),
        conditionVariable_(conditionVariable),
        stream_(ioContext_, K8SParams::SSLContext()),
        callback_(callback)
    {
        memset(&responseParser_, 0, sizeof(responseParser_));
        responseParser_.data = &responseParserState_;

        responseParserSetting_.on_headers_complete = [](http_parser* p) -> int
        {
            auto* state = static_cast<ResponseParserState* >(p->data);
            state->code_ = p->status_code;
            state->headComplete_ = true;
            return 0;
        };

        responseParserSetting_.on_body = [](http_parser* p, const char* at, size_t length) -> int
        {
            auto* state = static_cast<ResponseParserState* >(p->data);
            state->bodyBuffer_.append(at, length);
            state->bodyArrive_ = true;
            return 0;
        };

        responseParserSetting_.on_message_complete = [](http_parser* p) -> int
        {
            auto* state = static_cast<ResponseParserState* >(p->data);
            state->messageComplete_ = true;
            return 0;
        };
    }

    void start()
    {
        doConnect();
    }

 private:
    void afterError(const std::error_code& ec, const std::string& message) const
    {
        waitSyncCount_--;
        conditionVariable_.notify_all();
        if (callback_.onError)
        {
            return callback_.onError(ec, message);
        }
        std::cout << "K8SWatcher Error, Reason: " << ec.message() << ", Message: " << message << std::endl;
    }

    void clearResponseState()
    {
        responseParserState_.cleanup();
        http_parser_init(&responseParser_, HTTP_RESPONSE);
    }

    void doConnect()
    {
        const auto& apiServerIP = K8SParams::APIServerHost();
        if (apiServerIP.empty())
        {
            std::string msg = "fatal error: empty apiServerHost value";
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
        stream_.next_layer().async_connect(endpoint, [self](std::error_code ec)
        { self->afterConnect(ec); });
    }

    void afterConnect(const std::error_code& ec)
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
        stream_.async_handshake(asio::ssl::stream_base::client,
            [self](std::error_code ec)
            {
                self->afterHandshake(ec);
            });
    }

    void afterHandshake(const std::error_code& ec)
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
        requestContext_ = strStream.str();
        auto self = shared_from_this();
        asio::async_write(stream_, asio::buffer(requestContext_),
            [self](std::error_code ec, size_t transferred)
            { self->afterListRequest(ec, transferred); });
    }

    void afterListRequest(const std::error_code& ec, std::size_t transferred)
    {
        if (ec)
        {
            return afterError(ec, "error when write list request to apiServer");
        }
        clearResponseState();
        doReadListResponse();
    }

    void doReadListResponse()
    {
        auto self = shared_from_this();
        stream_.async_read_some(responseBuffer_.prepare(1024 * 1024 * 2),
            [self](std::error_code ec, size_t transferred)
            {
                self->afterReadListResponse(ec, transferred);
            });
    }

    void afterReadListResponse(const std::error_code& ec, size_t transferred)
    {
        if (ec)
        {
            return afterError(ec, "error when read list response from apiServer");
        }

        responseBuffer_.commit(transferred);

        const char* willParseData = asio::buffer_cast<const char*>(responseBuffer_.data());
        size_t parserLength = http_parser_execute(&responseParser_, &responseParserSetting_, willParseData, responseBuffer_.size());
        responseBuffer_.consume(parserLength);

        if (!responseParserState_.messageComplete_)
        {
            return doReadListResponse();
        }

        if (responseParserState_.code_ != HTTP_STATUS_OK)
        {
            return afterError(K8SWatcherError::UnexpectedResponse, responseParserState_.bodyBuffer_);
        }

        rapidjson::Document jsonDocument{};
        assert(!responseParserState_.bodyBuffer_.empty());
        jsonDocument.Parse(responseParserState_.bodyBuffer_.data(), responseParserState_.bodyBuffer_.size());

        if (jsonDocument.HasParseError())
        {
            return afterError(K8SWatcherError::UnexpectedResponse, responseParserState_.bodyBuffer_);
        }

        auto&& jsonArray = jsonDocument["items"].GetArray();
        if (callback_.onAdded)
        {
            for (auto&& item: jsonArray)
            {
                callback_.onAdded(item, K8SWatchEventDrive::List);
            }
        }
        auto pResourceVersion = rapidjson::GetValueByPointer(jsonDocument, "/metadata/resourceVersion");
        assert(pResourceVersion != nullptr && pResourceVersion->IsString());
        callback_.newestVersion_ = std::string(pResourceVersion->GetString(), pResourceVersion->GetStringLength());

        auto pContinue = rapidjson::GetValueByPointer(jsonDocument, "/metadata/continue");
        if (pContinue == nullptr)
        {
            callback_.continue_ = "";
        }
        else
        {
            assert(pContinue->IsString());
            callback_.continue_ = std::string(pContinue->GetString(), pContinue->GetStringLength());
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
        requestContext_ = strStream.str();
        auto self = shared_from_this();
        asio::async_write(stream_, asio::buffer(requestContext_),
            [self](std::error_code ec, size_t transferred)
            { self->afterWatchRequest(ec, transferred); });
    }

    void afterWatchRequest(const std::error_code& ec, std::size_t transferred)
    {
        if (ec)
        {
            return afterError(ec, "error when write watch request to apiServer");
        }
        clearResponseState();
        doReadWatchResponse();
    }

    void doReadWatchResponse()
    {
        auto self = shared_from_this();
        stream_.async_read_some(responseBuffer_.prepare(1024 * 1024 * 2),
            [self](std::error_code ec, size_t transferred)
            { self->afterReadWatchResponse(ec, transferred); });
    }

    void afterReadWatchResponse(const std::error_code& ec, size_t transferred)
    {
        if (ec)
        {
            return afterError(ec, "error when read watch response from apiServer");
        }
        responseBuffer_.commit(transferred);
        const char* responseData = asio::buffer_cast<const char*>(responseBuffer_.data());
        size_t parserSize = http_parser_execute(&responseParser_, &responseParserSetting_, responseData, responseBuffer_.size());

        responseBuffer_.consume(parserSize);

        if (!responseParserState_.headComplete_)
        {
            doReadWatchResponse();
            return;
        }

        assert(responseParserState_.headComplete_);

        constexpr unsigned int HTTP_OK = 200;

        if (responseParserState_.code_ != HTTP_OK)
        {
            if (!responseParserState_.messageComplete_)
            {
                doReadWatchResponse();
                return;
            }
            assert(responseParserState_.messageComplete_);
            return afterError(K8SWatcherError::UnexpectedResponse, responseParserState_.bodyBuffer_);
        }

        if (responseParserState_.bodyArrive_)
        {
            const char* bodyData = responseParserState_.bodyBuffer_.data();
            size_t bodyDataLength = responseParserState_.bodyBuffer_.size();

            rapidjson::StringStream stream(bodyData);
            size_t lastTell = 0;
            while (lastTell < bodyDataLength)
            {
                rapidjson::Document document{};
                document.ParseStream<rapidjson::kParseStopWhenDoneFlag>(stream);
                if (document.HasParseError())
                {
                    break;
                }
                stream.Take();
                lastTell = stream.Tell();
                if (!dispatchEvent(document))
                {
                    return;
                }
            }
            assert(lastTell <= bodyDataLength);
            size_t remainingLength = bodyDataLength - lastTell;
            responseParserState_.bodyBuffer_.replace(0, remainingLength, bodyData + lastTell);
            responseParserState_.bodyBuffer_.resize(remainingLength);
            responseParserState_.bodyArrive_ = false;
        }
        responseParserState_.messageComplete_ ? doWatchRequest() : doReadWatchResponse();
    }

    bool dispatchEvent(const rapidjson::Document& document)
    {
        assert(!document.HasParseError());

        auto pType = rapidjson::GetValueByPointer(document, "/type")->GetString();
        assert(pType != nullptr);

        constexpr char ADDEDTypeValue[] = "ADDED";
        constexpr char DELETETypeValue[] = "DELETED";
        constexpr char UPDATETypeValue[] = "MODIFIED";
        constexpr char BOOKMARKTypeValue[] = "BOOKMARK";
        constexpr char ERRORTypeValue[] = "ERROR";

        K8SWatchEvent what;

        if (strcmp(ADDEDTypeValue, pType) == 0)
        {
            what = K8SWatchEvent::K8SWatchEventAdded;
        }
        else if (strcmp(UPDATETypeValue, pType) == 0)
        {
            what = K8SWatchEvent::K8SWatchEventModified;
        }
        else if (strcmp(DELETETypeValue, pType) == 0)
        {
            what = K8SWatchEvent::K8SWatchEventDeleted;
        }
        else if (strcmp(BOOKMARKTypeValue, pType) == 0)
        {
            what = K8SWatchEvent::K8SWatchEventBookmark;
        }
        else if (strcmp(ERRORTypeValue, pType) == 0)
        {
            what = K8SWatchEvent::K8SWatchEventError;
        }
        else
        {
            return true;
        }

        if (what == K8SWatchEvent::K8SWatchEventError)
        {
            auto pErrorCode = rapidjson::GetValueByPointer(document, "/object/code");
            unsigned int errorCode = pErrorCode->GetUint();
            constexpr unsigned int HTTP_GONE = 410;
            if (errorCode == HTTP_GONE)
            {
                if (callback_.preList)
                {
                    callback_.preList();
                }
                doListRequest();
                return true;
            }
            afterError(K8SWatcherError::UnexpectedResponse, responseParserState_.bodyBuffer_);
            return false;
        }

        auto pResourceVersion = rapidjson::GetValueByPointer(document, "/object/metadata/resourceVersion");
        assert(pResourceVersion != nullptr && pResourceVersion->IsString());
        callback_.newestVersion_ = std::string(pResourceVersion->GetString(), pResourceVersion->GetStringLength());

        if (what == K8SWatchEvent::K8SWatchEventBookmark)
        {
            return true;
        }
        const auto& object = document["object"];
        switch (what)
        {
        case K8SWatchEvent::K8SWatchEventAdded:
            callback_.onAdded(object, K8SWatchEventDrive::Watch);
            return true;
        case K8SWatchEvent::K8SWatchEventDeleted:
            callback_.onDeleted(object);
            return true;
        case K8SWatchEvent::K8SWatchEventModified:
            callback_.onModified(object);
            return true;
        default:
            return true;
        }
        return true;
    }

 private:
    asio::io_context& ioContext_;
    std::atomic<int>& waitSyncCount_;
    std::condition_variable& conditionVariable_;
    asio::ssl::stream<asio::ip::tcp::socket> stream_;
    K8SWatcherSetting callback_;
    std::string requestContext_{};
    asio::streambuf responseBuffer_{};
    http_parser responseParser_{};
    http_parser_settings responseParserSetting_{};
    ResponseParserState responseParserState_{};
};

K8SWatcherSetting::K8SWatcherSetting(const std::string& group, const std::string& version, const std::string& plural, const std::string& _namespace)
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