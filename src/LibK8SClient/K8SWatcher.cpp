

#include "K8SWatcher.h"
#include <string>
#include <asio/io_context.hpp>
#include <thread>
#include <functional>
#include "rapidjson/document.h"
#include "K8SParams.h"
#include "HttpParser.h"
#include "asio/io_context.hpp"
#include "asio/ssl/stream.hpp"
#include "asio/streambuf.hpp"
#include "asio/ip/tcp.hpp"
#include "asio/error.hpp"
#include "rapidjson/document.h"
#include "rapidjson/pointer.h"
#include <string>
#include <stdexcept>
#include <utility>
#include <vector>
#include <map>
#include <memory>
#include <mutex>
#include <functional>
#include <iostream>
#include "condition_variable"

class K8SWatchSession : public std::enable_shared_from_this<K8SWatchSession> {
    struct ResponseParserState {

        ResponseParserState() = default;

        bool headComplete_{false};
        bool messageComplete_{false};
        bool bodyArrive_{false};
        unsigned int code_{0};
        std::string bodyBuffer_{};

        void cleanup() {
            headComplete_ = false;
            bodyArrive_ = false;
            messageComplete_ = false;
            bodyBuffer_.clear();
            code_=0;
        }
    };

public:
    K8SWatchSession(asio::io_context &ioContext, std::atomic<int> &waitCacheSyncCount, std::condition_variable &conditionVariable) :
            sslContext_(K8SParams::instance().sslContext()),
            waitCacheSyncCount_(waitCacheSyncCount),
            conditionVariable_(conditionVariable),
            stream_(ioContext, sslContext_) {

        memset(&responseParser_, 0, sizeof(responseParser_));
        responseParser_.data = &responseParserState_;

        responseParserSetting_.on_headers_complete = [](http_parser *p) -> int {
            auto *state = static_cast<ResponseParserState * >(p->data);
            state->code_ = p->status_code;
            state->headComplete_ = true;
            return 0;
        };

        responseParserSetting_.on_body = [](http_parser *p, const char *at, size_t length) -> int {
            auto *state = static_cast<ResponseParserState * >(p->data);
            state->bodyBuffer_.append(at, length);
            state->bodyArrive_ = true;
            return 0;
        };

        responseParserSetting_.on_message_complete = [](http_parser *p) -> int {
            auto *state = static_cast<ResponseParserState * >(p->data);
            state->messageComplete_ = true;
            return 0;
        };
    };

    void setCallBack(std::function<void(K8SWatchEvent, const rapidjson::Value &)> callBack) {
        _callBack = std::move(callBack);
    }

    void setResourceUrl(std::string url) {
        resourceUrl_ = std::move(url);
    }

    void run() {
        ++waitCacheSyncCount_;
        doConnect();
    }

private:

    void afterSocketFail(const asio::error_code &ec, const std::string &message) {
        std::cerr << "fatal error occurred while " << message << ", program will exit : " << ec.message();
        exit(-1);
    }

    void clearResponseState() {
        responseParserState_.cleanup();
        http_parser_init(&responseParser_, HTTP_RESPONSE);
    }

    void doConnect() {
        const auto &apiServerIP = K8SParams::instance().apiServerHost();
        if (apiServerIP.empty()) {
            std::cerr << "fatal error: empty apiServerHost value, program will exit" << std::endl;
            exit(-1);
        }
        const auto &apiServerPort = K8SParams::instance().apiServerPort();
        if (apiServerPort <= 0 || apiServerPort > 65535) {
            std::cerr << "fatal error: empty apiServerPort value, program will exit" << std::endl;
            exit(-1);
        }

        asio::ip::tcp::endpoint endpoint(asio::ip::address::from_string(apiServerIP), apiServerPort);
        auto self = shared_from_this();
        stream_.next_layer().async_connect(endpoint, [self](asio::error_code ec) { self->afterConnect(ec); });
    }

    void afterConnect(const asio::error_code &ec) {
        if (ec) {
            return afterSocketFail(ec, "connect to apiServer");
        }
        doHandleShake();
    };

    void doHandleShake() {
        auto self = shared_from_this();
        stream_.async_handshake(asio::ssl::stream_base::client,
                                [self](asio::error_code ec) {
                                    self->afterHandshake(ec);
                                });
    }

    void afterHandshake(const asio::error_code &ec) {
        if (ec) {
            return afterSocketFail(ec, "handshake with apiServer");
        }
        doListRequest();
    }

    void doListRequest() {
        std::ostringstream strStream;
        strStream << "GET ";
        strStream << resourceUrl_ << "?limit=30";
        if (!limitContinue_.empty()) {
            strStream << "&continue=" << limitContinue_;
        }
        strStream << " HTTP/1.1\r\n";
        strStream << "Host: " << K8SParams::instance().apiServerHost() << ":"
                  << K8SParams::instance().apiServerPort() << "\r\n";
        strStream << "Authorization: Bearer " << K8SParams::instance().bindToken() << "\r\n";
        strStream << "Connection: Keep-Alive\r\n";
        strStream << "\r\n";

        requestContext_ = strStream.str();

        auto self = shared_from_this();
        asio::async_write(stream_, asio::buffer(requestContext_), [self](asio::error_code ec, size_t bytes_transferred) { self->afterListRequest(ec, bytes_transferred); });
    }

    void afterListRequest(const asio::error_code &ec, std::size_t bytes_transferred) {
        if (ec) {
            return afterSocketFail(ec, "write list request to apiServer");
        }
        clearResponseState();
        doReadListResponse();
    }

    void doReadListResponse() {
        auto self = shared_from_this();
        stream_.async_read_some(responseBuffer_.prepare(1024 * 1024 * 2),
                                [self](asio::error_code ec, size_t bytes_transferred) {
                                    self->afterReadListResponse(ec, bytes_transferred);
                                });
    }

    void afterReadListResponse(const asio::error_code &ec, size_t bytes_transferred) {
        if (ec) {
            return afterSocketFail(ec, "read list response from apiServer");
        }

        responseBuffer_.commit(bytes_transferred);

        const char *willParseData = asio::buffer_cast<const char *>(responseBuffer_.data());
        size_t parserLength = http_parser_execute(&responseParser_, &responseParserSetting_, willParseData, responseBuffer_.size());
        responseBuffer_.consume(parserLength);

        if (!responseParserState_.messageComplete_) {
            return doReadListResponse();
        }

        if (responseParserState_.code_ != HTTP_STATUS_OK) {
            std::cerr << "k8sWatcher receive unexpected response, program will exit : \n\t" << responseParserState_.bodyBuffer_ << std::endl;
            exit(-1);
        }

        rapidjson::Document jsonDocument{};
        assert(!responseParserState_.bodyBuffer_.empty());
        jsonDocument.Parse(responseParserState_.bodyBuffer_.data(), responseParserState_.bodyBuffer_.size());

        if (jsonDocument.HasParseError()) {
            std::cerr << "k8sWatcher receive unexpected response, program will exit : \n\t" << responseParserState_.bodyBuffer_ << std::endl;
            exit(-1);
        }

        if (_callBack) {
            auto &&jsonArray = jsonDocument["items"].GetArray();
            for (auto &&item : jsonArray) {
                _callBack(K8SWatchEventAdded, item);
            }
        }

        auto pResourceVersion = rapidjson::GetValueByPointer(jsonDocument, "/metadata/resourceVersion");
        assert(pResourceVersion != nullptr);
        assert(pResourceVersion->IsString());
        auto iResourceVersion = std::stoull(pResourceVersion->GetString(), nullptr, 10);
        handledBiggestResourceVersion_ = std::max(handledBiggestResourceVersion_, iResourceVersion);
        auto pLimitContinue = rapidjson::GetValueByPointer(jsonDocument, "/metadata/continue");
        if (pLimitContinue == nullptr) {
            limitContinue_.clear();
            --waitCacheSyncCount_;
            conditionVariable_.notify_all();
            doWatchRequest();
            return;
        }
        assert(pLimitContinue->IsString());
        limitContinue_ = std::string(pLimitContinue->GetString(), pLimitContinue->GetStringLength());
        if (limitContinue_.empty()) {
            --waitCacheSyncCount_;
            conditionVariable_.notify_all();
            doWatchRequest();
            return;
        };
        doListRequest();
    }

    void doWatchRequest() {
        std::ostringstream strStream;
        strStream << "GET ";
        strStream << resourceUrl_ << "?watch=1&allowWatchBookmarks=true&timeoutSeconds=" << (std::rand() % 14 + 45) * 60;
        if (handledBiggestResourceVersion_ != 0) {
            strStream << "&resourceVersion=" << handledBiggestResourceVersion_;
        }
        strStream << " HTTP/1.1\r\n";
        strStream << "Host: " << K8SParams::instance().apiServerHost() << ":"
                  << K8SParams::instance().apiServerPort() << "\r\n";
        strStream << "Authorization: Bearer " << K8SParams::instance().bindToken() << "\r\n";
        strStream << "Connection: Keep-Alive\r\n";
        strStream << "\r\n";

        requestContext_ = strStream.str();
        auto self = shared_from_this();
        asio::async_write(stream_, asio::buffer(requestContext_), [self](asio::error_code ec, size_t bytes_transferred) { self->afterWatchRequest(ec, bytes_transferred); });
    }

    void afterWatchRequest(const asio::error_code &ec, std::size_t bytes_transferred) {
        if (ec) {
            return afterSocketFail(ec, "write watch request to apiServer");
        }
        clearResponseState();
        doReadWatchResponse();
    }

    void doReadWatchResponse() {
        stream_.async_read_some(responseBuffer_.prepare(1024 * 1024 * 2),
                                std::bind(&K8SWatchSession::afterReadWatchResponse, shared_from_this(),
                                          std::placeholders::_1, std::placeholders::_2));
    }

    void afterReadWatchResponse(const asio::error_code &ec, size_t bytes_transferred) {
        if (ec) {
            return afterSocketFail(ec, "read watch response from apiServer");
        }
        responseBuffer_.commit(bytes_transferred);

        const char *responseData = asio::buffer_cast<const char *>(responseBuffer_.data());
        size_t parserSize = http_parser_execute(&responseParser_, &responseParserSetting_, responseData, responseBuffer_.size());
        assert(parserSize == bytes_transferred);
        responseBuffer_.consume(parserSize);

        if (!responseParserState_.headComplete_) {
            doReadWatchResponse();
            return;
        }

        assert(responseParserState_.headComplete_);

        constexpr unsigned int HTTP_OK = 200;

        if (responseParserState_.code_ != HTTP_OK) {
            if (!responseParserState_.messageComplete_) {
                doReadWatchResponse();
                return;
            }
            assert(responseParserState_.messageComplete_);
            std::cerr << "k8sWatcher receive unexpected response, program will exit : \n\t" << responseParserState_.bodyBuffer_ << std::endl;
            exit(-1);
        }

        if (responseParserState_.bodyArrive_) {

            const char *bodyData = responseParserState_.bodyBuffer_.data();
            size_t bodyDataLength = responseParserState_.bodyBuffer_.size();

            rapidjson::StringStream stream(bodyData);
            size_t lastTell = 0;
            while (lastTell < bodyDataLength) {
                rapidjson::Document document{};
                document.ParseStream<rapidjson::kParseStopWhenDoneFlag>(stream);
                if (document.HasParseError()) {
                    break;
                }
                stream.Take();
                lastTell = stream.Tell();
                if (!handlerK8SWatchEvent(document)) {
                    std::cerr << "k8sWatcher receive unexpected response, program will exit : \n\t" << responseParserState_.bodyBuffer_ << std::endl;
                    exit(-1);
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

    bool handlerK8SWatchEvent(const rapidjson::Document &document) {
        assert(!document.HasParseError());

        auto pType = rapidjson::GetValueByPointer(document, "/type")->GetString();
        assert(pType != nullptr);

        constexpr char ADDEDTypeValue[] = "ADDED";
        constexpr char DELETETypeValue[] = "DELETED";
        constexpr char UPDATETypeValue[] = "MODIFIED";
        constexpr char BOOKMARKTypeValue[] = "BOOKMARK";
        constexpr char ERRORTypeValue[] = "ERROR";

        K8SWatchEvent what;

        if (strcmp(ADDEDTypeValue, pType) == 0) {
            what = K8SWatchEventAdded;
        } else if (strcmp(UPDATETypeValue, pType) == 0) {
            what = K8SWatchEventUpdate;
        } else if (strcmp(DELETETypeValue, pType) == 0) {
            what = K8SWatchEventDeleted;
        } else if (strcmp(BOOKMARKTypeValue, pType) == 0) {
            what = K8SWatchEventBookmark;
        } else if (strcmp(ERRORTypeValue, pType) == 0) {
            what = K8SWatchEventError;
        } else {
            std::cerr << "k8sWatcher receive unknown event type : " << pType << std::endl;
            return true;
        }

        if (what == K8SWatchEventError) {
            auto pErrorCode = rapidjson::GetValueByPointer(document, "/object/code");
            unsigned int errorCode = pErrorCode->GetUint();
            constexpr unsigned int HTTP_GONE = 410;
            if (errorCode == HTTP_GONE) {
                auto pMessageJson = rapidjson::GetValueByPointer(document, "/object/message");
                assert(pMessageJson != nullptr);
                auto pMessage = pMessageJson->GetString();
                assert(pMessage != nullptr);
                auto begin = strchr(pMessage, '(');
                if (begin != nullptr) {
                    auto v = std::stoull(begin + 1, nullptr);
                    handledBiggestResourceVersion_ = std::max(handledBiggestResourceVersion_, v);
                } else {
                    handledBiggestResourceVersion_ += 100;
                    //fixme 目前采用当前 handledBiggestResourceVersion_+=100,来处理 Gone 的情况,
                    // 但并不足够稳妥. 应该获取到资源对应的 --watch-cache-sizes 后再处理
                }
                return true;
            }
            return false;
        }

        auto pResourceVersion = rapidjson::GetValueByPointer(document, "/object/metadata/resourceVersion");
        assert(pResourceVersion != nullptr);
        assert(pResourceVersion->IsString());
        auto iResourceVersion = std::stoull(pResourceVersion->GetString(), nullptr, 10);

        if (iResourceVersion >= handledBiggestResourceVersion_) {
            if (what != K8SWatchEventBookmark) {
                if (_callBack) {
                    const auto &item = document["object"];
                    _callBack(what, item);
                }
            }
            handledBiggestResourceVersion_ = iResourceVersion;
        }
        return true;
    }

private:
    asio::ssl::context &sslContext_;
    std::atomic<int> &waitCacheSyncCount_;
    std::condition_variable &conditionVariable_;

    unsigned long long handledBiggestResourceVersion_{0};
    asio::ssl::stream <asio::ip::tcp::socket> stream_;

    std::string resourceUrl_;
    std::string limitContinue_{};

    std::string requestContext_{};
    asio::streambuf responseBuffer_{};

    http_parser responseParser_{};
    http_parser_settings responseParserSetting_{};
    ResponseParserState responseParserState_{};

    std::function<void(K8SWatchEvent, const rapidjson::Value &)> _callBack;
};

void K8SWatcher::postWatch(const std::string &url, const std::function<void(K8SWatchEvent, const rapidjson::Value &)> &callback) {
    auto session = std::make_shared<K8SWatchSession>(ioContext_, waitCacheSyncCount_, conditionVariable_);
    session->setCallBack(callback);
    session->setResourceUrl(url);
    session->run();
}

void K8SWatcher::waitForCacheSync() {
    std::unique_lock <std::mutex> uniqueLock(mutex_);
    auto finish = conditionVariable_.wait_for(uniqueLock, std::chrono::seconds(120), [this]() {
        return waitCacheSyncCount_ == 0;
    });
    if (!finish) {
        std::cerr << "k8sWatcher wait cache sync overtime, program will exit" << std::endl;
        exit(-1);
    }
}
