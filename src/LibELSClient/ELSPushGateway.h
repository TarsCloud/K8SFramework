#pragma once

#include <thread>
#include <mutex>
#include <random>
#include <vector>
#include <asio/io_context.hpp>
#include <asio/ip/tcp.hpp>
#include <functional>

class ELSIndexEntry;

class ELKPushGateway {
    friend class ELSIndexEntry;

public:
    static ELKPushGateway &instance() {
        static ELKPushGateway gateway;
        return gateway;
    }

public:

    void setELKNodeAddress(std::vector<std::tuple<std::string, int>> elkNodeAddresses);

    void postData(const std::string &index, std::string &&data);

    void postData(const std::string &index, std::vector<std::string> &&data);

    void start();

    void updateSyncDuration(std::size_t syncDuration);

    inline void setFailCallback(std::function<void(const std::string &)> callback) {
        _failCallback = std::move(callback);
    }

private:

    ELKPushGateway() = default;

    void getELKNodeAddress(std::string &host, int &port);

    inline void notifyFail(const std::string &message) {
        if (_failCallback) {
            _failCallback(message);
        }
    }

private:
    asio::io_context _ioContext{1};
    std::thread _thread{};
    std::mutex _mutex{};
    std::shared_ptr<ELSIndexEntry> _indexEntry;
    std::vector<std::tuple<std::string, int>> _elkNodeAddresses{};
    std::function<void(const std::string &message)> _failCallback;
};
