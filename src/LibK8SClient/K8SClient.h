
#pragma once


#include <mutex>
#include <thread>
#include <queue>
#include <memory>
#include <vector>
#include <condition_variable>
#include <boost/json.hpp>
#include <boost/asio/io_context.hpp>
#include <boost/asio/posix/stream_descriptor.hpp>

enum class K8SClientRequestMethod
{
    Patch,
    Post,
    StrategicMergePatch,
    Delete,
    Get,
};

enum K8SClientRequestState
{
    Pending,
    Running,
    Done,
    Error,
    Cancel,
};

class K8SClientRequestEntry;

class K8SClientRequest
{
    friend class K8SClient;

    friend class K8SClientWorker;

public:
    void waitFinish()
    {
        std::unique_lock <std::mutex> uniqueLock(mutex_);
        return conditionVariable_.wait(uniqueLock, [this]()
        {
            return state_ == Error || state_ == Done;
        });
    }

    template<typename Rep, typename Period>
    bool waitFinish(const std::chrono::duration <Rep, Period>& time)
    {
        std::unique_lock <std::mutex> uniqueLock(mutex_);
        return conditionVariable_.wait_for(uniqueLock, time, [this]()
        {
            return state_ == Error || state_ == Done;
        });
    }

    const std::atomic<int>& state() const
    {
        return state_;
    }

    const std::string& stateMessage() const
    {
        return stateMessage_;
    }

    const char* responseBody() const;

    size_t responseSize() const;

    const boost::json::value& responseJson() const;

    unsigned int responseCode() const;

private:
    void setState(int state, std::string stateMessage)
    {
        std::unique_lock <std::mutex> uniqueLock(mutex_);
        state_ = state;
        stateMessage_.swap(stateMessage);
        conditionVariable_.notify_all();
    }

private:

    K8SClientRequest() = default;

    std::atomic<int> state_{ K8SClientRequestState::Pending };
    std::string stateMessage_;
    std::mutex mutex_;
    std::condition_variable conditionVariable_;
    std::shared_ptr <K8SClientRequestEntry> entry_;
};

class K8SClientWorker;

class K8SClient
{
public:
    static K8SClient& instance()
    {
        static K8SClient k8SClient;
        return k8SClient;
    }

    K8SClient() = default;

    ~K8SClient()
    {
        ioContext_.stop();
    }

    void start();

    void stop();

    std::shared_ptr <K8SClientRequest>
    postRequest(K8SClientRequestMethod method, const std::string& url, const std::string& body);

private:
    static std::string buildPostRequest(const std::string& url, const std::string& body);

    static std::string buildPatchRequest(const std::string& url, const std::string& body);

    static std::string buildSMPatchRequest(const std::string& url, const std::string& body);

    static std::string buildDeleteRequest(const std::string& url);

    static std::string buildGetRequest(const std::string& url);

private:
    boost::asio::io_context ioContext_{ 1 };
    boost::asio::posix::stream_descriptor pipeReadStream_{ ioContext_ };
    boost::asio::posix::stream_descriptor pipeWriteStream_{ ioContext_ };
    std::queue <std::shared_ptr<K8SClientRequest>> pendingQueue_{};
    std::vector <std::shared_ptr<K8SClientWorker>> sessionVector_{};
    std::thread thread_;
};
