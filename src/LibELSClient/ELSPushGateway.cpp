#include "ELSPushGateway.h"
#include "HttpParser.h"
#include "asio/streambuf.hpp"
#include "asio/steady_timer.hpp"
#include <iostream>
#include <utility>
#include <queue>
#include <atomic>

class ELSHTTPEntry {
public:
    ELSHTTPEntry() {
        memset(&responseParserSetting_, 0, sizeof(responseParserSetting_));
        responseParserSetting_.on_body = ([](http_parser *p, const char *d, size_t l) -> int {
            auto *entry = static_cast<ELSHTTPEntry *const>(p->data);
            entry->httpResponseBody_.append(d, l);
            return 0;
        });

        responseParserSetting_.on_message_complete = [](http_parser *p) -> int {
            auto *entry = static_cast<ELSHTTPEntry *const>(p->data);
            entry->httpResponseComplete_ = true;
            return 0;
        };

        responseClear();
    }

    void allClear() {
        httpRequestHeader_.clear();
        httpRequestBody_.clear();
        httpRequestHost_.clear();
        httpRequestPort_ = {};
        responseClear();
    }

    void responseClear() {
        memset(&httpResponseParser_, 0, sizeof(httpResponseParser_));
        http_parser_init(&httpResponseParser_, HTTP_RESPONSE);
        httpResponseParser_.data = this;
        httpResponseBody_.clear();
        httpResponseComplete_ = false;
    }

    bool httpResponseComplete_{false};
    int httpRequestPort_{};
    std::string httpRequestHost_{};
    std::string httpRequestHeader_{};
    std::string httpRequestBody_{};
    std::string httpResponseBody_{};
    http_parser httpResponseParser_{};
    http_parser_settings responseParserSetting_{};
};

class ELSIndexEntry : public std::enable_shared_from_this<ELSIndexEntry> {
public:
    // if waitingSyncQueue.size() >= CACHE_SIZE_HARD_LIMIT , header data of the queue will be discard;
    static size_t CACHE_SIZE_HARD_LIMIT;

    static size_t CACHE_SIZE_SOFT_LIMIT;

    static size_t MAX_CACHE_SYNC_DURATION;

    static size_t MIN_CACHE_SYNC_DURATION;

    static size_t DEFAULT_CACHE_SYNC_DURATION;

    static size_t MAX_RETRY_TIMES;

    explicit ELSIndexEntry(std::string indexName, asio::io_context &ioContext)
            : indexName_(std::move(indexName)),
              ioContext_(ioContext) {
    }

    void start() {
        waitNotify();
    };

    void postData(const std::shared_ptr<std::string> &dataPtr) {
        auto &&data = *dataPtr;
        waitingSyncQueue_.emplace(std::move(data));
        if (waitingSyncQueue_.size() > CACHE_SIZE_HARD_LIMIT) {
            for (auto i = waitingSyncQueue_.size() - CACHE_SIZE_HARD_LIMIT; i > 0; --i) {
                waitingSyncQueue_.pop();
            }
        }
    }

    void postData(const std::shared_ptr<std::vector<std::string>> &dataPtr) {
        auto &&dataVec = *dataPtr;
        if (waitingSyncQueue_.size() + dataVec.size() > CACHE_SIZE_HARD_LIMIT) {
            for (auto i = 0; i < waitingSyncQueue_.size() + dataVec.size() - CACHE_SIZE_HARD_LIMIT; ++i) {
                waitingSyncQueue_.pop();
            }
        }
        for (auto &&item : dataVec) {
            waitingSyncQueue_.emplace(std::move(item));
        }
    }

    void release() {
        shouldExited_ = true;
    }

    const std::string &name() {
        return indexName_;
    }

private:

    void retry() {
        auto self = shared_from_this();
        auto timer = std::make_shared<asio::steady_timer>(ioContext_);
        auto steadyTime = std::min((1ul << retryTimes_) + 10ul, 1000 * 30ul);
        timer->expires_after(std::chrono::milliseconds(steadyTime));
        timer->async_wait([timer, self](asio::error_code ec) {
            self->doCreateStreamThenSyncES();
        });
    }

    void afterSocketError() {
        std::ostringstream errorMessageStream;
        errorMessageStream << "write to es" << "(" << httpEntry_.httpRequestHost_ << ":" << httpEntry_.httpRequestPort_
                           << ") fail, retry times  " << retryTimes_;
        std::string errorMessage = errorMessageStream.str();
        ELKPushGateway::instance().notifyFail(errorMessage);
        std::cout << errorMessage << std::endl;
        ++retryTimes_;
        retryTimes_ <= MAX_RETRY_TIMES ? retry() : waitNotify();
    }

    void waitNotify() {
        auto self = shared_from_this();
        auto timer = std::make_shared<asio::steady_timer>(ioContext_);
        timer->expires_after(std::chrono::milliseconds(DEFAULT_CACHE_SYNC_DURATION));
        timer->async_wait([timer, self](asio::error_code ec) {
            self->afterNotify(ec);
        });
    }

    void afterNotify(const asio::error_code &ec) {
        doPrepareSync();
    }

    void doPrepareSync() {
        if (waitingSyncQueue_.empty()) {
            if (!shouldExited_) {
                waitNotify();
            }
            return;
        }
        httpEntry_.allClear();
        for (auto i = 0; i < CACHE_SIZE_SOFT_LIMIT; ++i) {
            if (waitingSyncQueue_.empty()) {
                break;
            }
            auto &&item = waitingSyncQueue_.front();
            httpEntry_.httpRequestBody_.append(R"({"index":{"_id":")").append(generaId(indexName_, item)).append(
                    "\"}}\n");
            httpEntry_.httpRequestBody_.append(item).append("\n");
            waitingSyncQueue_.pop();
        }
        httpEntry_.httpRequestHeader_.append("POST /").append(indexName_).append("/_bulk").append(" HTTP/1.1\r\n");
        httpEntry_.httpRequestHeader_.append("Content-Type: application/x-ndjson\r\n");
        httpEntry_.httpRequestHeader_.append("Connection: Keep-Alive\r\n");
        httpEntry_.httpRequestHeader_.append("Content-Length: ").append(
                std::to_string(httpEntry_.httpRequestBody_.size())).append("\r\n");
        httpEntry_.httpRequestHeader_.append("\r\n");
        if (socketStream_ == nullptr || !socketStream_->is_open()) {
            doCreateStreamThenSyncES();
            return;
        }
        doSyncES();
    }

    void doCreateStreamThenSyncES() {
        doResolve();
    }

    void doResolve() {
        auto resolver = std::make_shared<asio::ip::tcp::resolver>(ioContext_);
        auto self = shared_from_this();
        ELKPushGateway::instance().getELKNodeAddress(httpEntry_.httpRequestHost_, httpEntry_.httpRequestPort_);
        resolver->async_resolve(asio::ip::tcp::v4(), httpEntry_.httpRequestHost_,
                                std::to_string(httpEntry_.httpRequestPort_),
                                [self, resolver](asio::error_code ec,
                                                 const asio::ip::tcp::resolver::results_type &results) {
                                    self->afterResolve(ec, results);
                                });
    }

    void afterResolve(const asio::error_code &ec, const asio::ip::tcp::resolver::results_type &results) {
        if (ec) {
            afterSocketError();
            return;
        }
        if (results.empty()) {
            afterSocketError();
            return;
        }
        doConnect(results);
    }

    void doConnect(const asio::ip::tcp::resolver::results_type &results) {
        auto self = shared_from_this();
        socketStream_ = std::make_shared<asio::ip::tcp::socket>(ioContext_);
        socketStream_->async_connect(*results, [self](asio::error_code ec) {
            self->afterConnect(ec);
        });
    }

    void afterConnect(const asio::error_code &ec) {
        if (ec) {
            afterSocketError();
            return;
        }
        doSyncES();
    }

    void doSyncES() {
        doWriteElSRequestHead();
    }

    void doWriteData(const char *data, size_t len, const std::function<void(asio::error_code)> &cb) {
        auto self = shared_from_this();
        socketStream_->async_write_some(asio::buffer(data, len),
                                        [self, data, len, cb](asio::error_code ec, size_t bytes_transferred) {
                                            if (ec) {
                                                cb(ec);
                                                return;
                                            }
                                            assert(len >= bytes_transferred);
                                            if (bytes_transferred == len) {
                                                cb(asio::error_code());
                                                return;
                                            }
                                            if (bytes_transferred < len) {
                                                self->doWriteData(data + bytes_transferred, len - bytes_transferred,
                                                                  cb);
                                                return;
                                            }
                                        });
    };

    void doWriteElSRequestHead() {
        auto self = shared_from_this();
        doWriteData(httpEntry_.httpRequestHeader_.data(), httpEntry_.httpRequestHeader_.size(),
                    [self](asio::error_code ec) { self->afterWritELSRequestHead(ec); });
    }

    void afterWritELSRequestHead(const asio::error_code &ec) {
        if (ec) {
            afterSocketError();
            return;
        }
        doWriteElSRequestBody();
    }

    void doWriteElSRequestBody() {
        auto self = shared_from_this();
        doWriteData(httpEntry_.httpRequestBody_.data(), httpEntry_.httpRequestBody_.size(),
                    [self](asio::error_code ec) { self->afterWriteELSRequestBody(ec); });
    }

    void afterWriteELSRequestBody(const asio::error_code &ec) {
        if (ec) {
            afterSocketError();
            return;
        }
        doReadELSResponse();
    }

    void doReadELSResponse() {
        auto self = shared_from_this();
        socketStream_->async_read_some(responseBuffer_.prepare(1024 * 1024 * 4),
                                       [self](asio::error_code ec, size_t bytes_transferred) {
                                           self->afterReadELSResponse(ec, bytes_transferred);
                                       });
    }

    void afterReadELSResponse(const asio::error_code &ec, size_t bytes_transferred) {
        if (ec) {
            afterSocketError();
            return;
        }

        responseBuffer_.commit(bytes_transferred);
        const char *willParseData = asio::buffer_cast<const char *>(responseBuffer_.data());
        const auto willParserLength = responseBuffer_.size();
        size_t parserLength = http_parser_execute(&httpEntry_.httpResponseParser_, &httpEntry_.responseParserSetting_,
                                                  willParseData, willParserLength);
        if (parserLength != bytes_transferred) {
            afterSocketError();
            return;
        }

        responseBuffer_.consume(parserLength);

        if (!httpEntry_.httpResponseComplete_) {
            doReadELSResponse();
            return;
        }
        constexpr uint HTTP_OK_CODE = 200;
        if (httpEntry_.httpResponseParser_.status_code != HTTP_OK_CODE) {
            afterSocketError();
            return;
        }
        //todo check one record fail;
        httpEntry_.allClear();
        retryTimes_ = 0;
        doPrepareSync();
    }

    static std::string generaId(const std::string &indexName, const std::string &context) {
        static std::atomic<uint64_t> seq{1ull << 63u};
        seq++;
        auto iContentHash = std::hash<std::string>{}(context);
        auto iIndexNameHash = std::hash<std::string>{}(indexName);
        auto hashValue = (iContentHash << 48u) + (iIndexNameHash << 16u) + seq;
        return std::to_string(hashValue);
    };

private:
    const std::string indexName_;
    size_t retryTimes_{};
    asio::io_context &ioContext_;
    std::shared_ptr<asio::ip::tcp::socket> socketStream_;
    std::queue<std::string> waitingSyncQueue_;
    bool shouldExited_{false};
    ELSHTTPEntry httpEntry_;
    asio::streambuf responseBuffer_{};
};

size_t ELSIndexEntry::CACHE_SIZE_HARD_LIMIT = 300000;  // About 120MB Memory
size_t ELSIndexEntry::CACHE_SIZE_SOFT_LIMIT = 500;
size_t ELSIndexEntry::MAX_CACHE_SYNC_DURATION = 1000 * 60 * 1; //1 minutes
size_t ELSIndexEntry::MIN_CACHE_SYNC_DURATION = 100; // 100 ms
size_t ELSIndexEntry::DEFAULT_CACHE_SYNC_DURATION = 500; // 500 ms
size_t ELSIndexEntry::MAX_RETRY_TIMES = 30;

void ELKPushGateway::updateSyncDuration(size_t syncDuration) {
    if (syncDuration < ELSIndexEntry::MIN_CACHE_SYNC_DURATION) {
        ELSIndexEntry::DEFAULT_CACHE_SYNC_DURATION = ELSIndexEntry::MIN_CACHE_SYNC_DURATION;
    } else if (syncDuration > ELSIndexEntry::MAX_CACHE_SYNC_DURATION) {
        ELSIndexEntry::DEFAULT_CACHE_SYNC_DURATION = ELSIndexEntry::MAX_CACHE_SYNC_DURATION;
    } else {
        ELSIndexEntry::DEFAULT_CACHE_SYNC_DURATION = syncDuration;
    }
}

void ELKPushGateway::setELKNodeAddress(std::vector<std::tuple<std::string, int>> elkNodeAddresses) {
    if (elkNodeAddresses.empty()) {
        throw std::runtime_error(
                std::string("fatal error: empty elk node addresses"));
    }

    for (auto &&tuple:elkNodeAddresses) {
        auto &&port = std::get<1>(tuple);
        if (port > 65535 || port <= 0) {
            throw std::runtime_error(
                    std::string("fatal error: bad elk node port value/").append(std::to_string(port)));
        }
        auto &&host = std::get<0>(tuple);
        if (host.empty()) {
            throw std::runtime_error(
                    std::string("fatal error: bad elk node host value/").append(host));
        }
    }

    std::lock_guard<std::mutex> lockGuard(_mutex);
    _elkNodeAddresses = std::move(elkNodeAddresses);
}

void ELKPushGateway::postData(const std::string &index, std::string &&data) {
    std::lock_guard<std::mutex> lockGuard(_mutex);
    if (_indexEntry == nullptr) {
        _indexEntry = std::make_shared<ELSIndexEntry>(index, _ioContext);
        _indexEntry->start();
    } else if (_indexEntry->name() != index) {
        _indexEntry->release();
        _indexEntry = std::make_shared<ELSIndexEntry>(index, _ioContext);
        _indexEntry->start();
    }

    auto dataPtr = std::make_shared<std::string>(std::move(data));
    _ioContext.post([this, dataPtr] { _indexEntry->postData(dataPtr); });
}

void ELKPushGateway::postData(const std::string &index, std::vector<std::string> &&data) {
    std::lock_guard<std::mutex> lockGuard(_mutex);
    if (_indexEntry == nullptr) {
        _indexEntry = std::make_shared<ELSIndexEntry>(index, _ioContext);
        _indexEntry->start();
    } else if (_indexEntry->name() != index) {
        _indexEntry->release();
        _indexEntry = std::make_shared<ELSIndexEntry>(index, _ioContext);
        _indexEntry->start();
    }

    auto dataPtr = std::make_shared<std::vector<std::string>>(std::move(data));
    _ioContext.post([this, dataPtr] { _indexEntry->postData(dataPtr); });
}

void ELKPushGateway::start() {
    _thread = std::thread([this] {
        asio::io_context::work work(_ioContext);
        _ioContext.run();
    });
    _thread.detach();
}

void ELKPushGateway::getELKNodeAddress(std::string &host, int &port) {
    std::lock_guard<std::mutex> lockGuard(_mutex);
    if (_elkNodeAddresses.empty()) {
        throw std::runtime_error(
                std::string("fatal error: empty elk node addresses"));
    }
    auto tuple = _elkNodeAddresses[std::rand() % _elkNodeAddresses.size()];
    host = std::get<0>(tuple);
    port = std::get<1>(tuple);
}
