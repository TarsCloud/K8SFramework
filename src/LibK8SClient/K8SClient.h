
#pragma once

#include <memory>
#include <vector>
#include <asio/ssl/stream.hpp>
#include <asio/ip/tcp.hpp>
#include <asio/streambuf.hpp>
#include <asio/write.hpp>
#include <asio/read.hpp>
#include <asio/read_until.hpp>
#include <asio/connect.hpp>
#include <queue>
#include <mutex>
#include <iostream>
#include <chrono>
#include <atomic>
#include <thread>
#include <asio/posix/stream_descriptor.hpp>
#include <condition_variable>
#include "HttpParser.h"
#include "rapidjson/document.h"

enum class K8SClientRequestMethod {
    Patch,
    Post,
    StrategicMergePatch,
    Delete,
    Get,
};

enum K8SClientRequestState {
    Pending,
    Running,
    Done,
    Error,
    Cancel,
};

class K8SClientRequestEntry;

class K8SClientRequest {
    friend class K8SClient;

    friend class K8SClientWorker;

public:
    void waitFinish() {
        std::unique_lock <std::mutex> uniqueLock(mutex_);
        return conditionVariable_.wait(uniqueLock, [this]() {
            return state_ == Error || state_ == Done;
        });
    }

    template<typename Rep, typename Period>
    bool waitFinish(const std::chrono::duration <Rep, Period> &time) {
        std::unique_lock <std::mutex> uniqueLock(mutex_);
        return conditionVariable_.wait_for(uniqueLock, time, [this]() {
            return state_ == Error || state_ == Done;
        });
    }

    const std::atomic<int> &state() const {
        return state_;
    }

    const std::string &stateMessage() const {
        return stateMessage_;
    }

    const char *responseBody() const;

    size_t responseSize() const;

    const rapidjson::Value &responseJson() const;

    unsigned int responseCode() const;

private:
    void setState(int state, std::string stateMessage) {
        std::unique_lock <std::mutex> uniqueLock(mutex_);
        state_ = state;
        stateMessage_.swap(stateMessage);
        conditionVariable_.notify_all();
    }

private:

    K8SClientRequest() = default;

    std::atomic<int> state_{K8SClientRequestState::Pending};
    std::string stateMessage_;
    std::mutex mutex_;
    std::condition_variable conditionVariable_;
    std::shared_ptr <K8SClientRequestEntry> entry_;
};

class K8SClientWorker;

class K8SClient {
public:
    static K8SClient &instance() {
        static K8SClient k8SClient;
        return k8SClient;
    }

    void start();

    std::shared_ptr <K8SClientRequest> postRequest(K8SClientRequestMethod method, const std::string &url, const std::string &body);

private:
    static std::string buildPostRequest(const std::string &url, const std::string &body);

    static std::string buildPatchRequest(const std::string &url, const std::string &body);

    static std::string buildSMPatchRequest(const std::string &url, const std::string &body);

    static std::string buildDeleteRequest(const std::string &url);

    static std::string buildGetRequest(const std::string &url);

    K8SClient() = default;

private:
    asio::io_context ioContext_{1};
    asio::posix::stream_descriptor eventStream_{ioContext_};
    std::queue <std::shared_ptr<K8SClientRequest>> pendingQueue_{};
    std::vector <std::shared_ptr<K8SClientWorker>> sessionVector_{};
    std::thread thread_;
};
