
#pragma once

#include <thread>
#include <functional>
#include <rapidjson/document.h>
#include <atomic>
#include <mutex>
#include <condition_variable>
#include <asio/io_context.hpp>

enum K8SWatchEvent {
    K8SWatchEventAdded = 1u,
    K8SWatchEventDeleted = 2u,
    K8SWatchEventUpdate = 3u,
    K8SWatchEventBookmark = 4u,
    K8SWatchEventError = 5u,
};

class K8SWatcher {
public:
    static K8SWatcher &instance() {
        static K8SWatcher k8SWatcher;
        return k8SWatcher;
    }

    void start() {
        thread_ = std::thread([this] {
            asio::io_context::work work(ioContext_);
            ioContext_.run();
        });
        thread_.detach();
        waitForCacheSync();
    }

    void postWatch(const std::string &url, const std::function<void(K8SWatchEvent, const rapidjson::Value &)> &);

private:
    void waitForCacheSync();

private:
    std::atomic<int> waitCacheSyncCount_{};
    std::mutex mutex_;
    std::condition_variable conditionVariable_;
    asio::io_context ioContext_;
    std::thread thread_;
};
