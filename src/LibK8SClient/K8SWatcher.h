#pragma once


#include <mutex>
#include <string>
#include <thread>
#include <functional>
#include <condition_variable>
#include <boost/json.hpp>
#include <boost/asio/io_context.hpp>

enum class K8SWatchEventDrive
{
    List,
    Watch
};

class K8SWatcherSetting
{
    friend class K8SWatcherSession;

public:
    K8SWatcherSetting(const std::string& group, const std::string& version, const std::string& plural,
            const std::string& _namespace);;

    K8SWatcherSetting(const K8SWatcherSetting&) = default;

    ~K8SWatcherSetting() = default;

    std::function<void()> preList;
    std::function<void()> postList;
    std::function<void(const boost::json::value&, K8SWatchEventDrive)> onAdded;
    std::function<void(const boost::json::value&)> onDeleted;
    std::function<void(const boost::json::value&)> onModified;
    std::function<bool(const boost::system::error_code&, const std::string&)> onError;;

    std::string watchUri() const;

    std::string listUri() const;

    void setLabelFilter(std::string filter)
    {
        labelFilter_ = std::move(filter);
    }

    void setFiledFilter(std::string filter)
    {
        filedFilter_ = std::move(filter);
    }

    void setLimit(size_t limit)
    {
        limit_ = limit;
    }

private:
    size_t limit_ = { 30 };
    size_t overtime_ = { 60 * 30 };
    std::string labelFilter_{};
    std::string filedFilter_{};
    std::string newestVersion_{};
    std::string path_;
    std::string continue_;
};

class K8SWatcher
{
public:
    static K8SWatcher& instance()
    {
        static K8SWatcher k8SWatcher;
        return k8SWatcher;
    }

    void start();

    void stop();

    template<typename Rep, typename Period>
    bool waitSync(const std::chrono::duration<Rep, Period>& time)
    {
        std::unique_lock<std::mutex> uniqueLock(mutex_);
        return conditionVariable_.wait_for(uniqueLock, time, [this]()
        {
            return waitSyncCount_ == 0;
        });
    }

    void addWatch(const K8SWatcherSetting& setting);

    ~K8SWatcher()
    {
        ioContext_.stop();
    }

private:
    K8SWatcher() = default;

private:
    boost::asio::io_context ioContext_{ 1 };
    std::atomic_int waitSyncCount_{ 0 };
    std::mutex mutex_;
    std::condition_variable conditionVariable_;
    std::thread thread_;
};
