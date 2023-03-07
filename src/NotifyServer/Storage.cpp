
#include "Storage.h"
#include "util/tc_thread_rwlock.h"
#include <string>
#include "TCodec.h"

class StorageImp
{
public:
    static StorageImp& instance()
    {
        static StorageImp imp;
        return imp;
    }

    StorageImp(const StorageImp&) = delete;

    ~StorageImp() = default;

private:
    StorageImp() = default;

public:
    tars::TC_ThreadRWLocker mutex_;
    std::unordered_map<std::string, std::string> map_;
    std::unordered_map<std::string, std::string> cache_;
};

void Storage::onPodAdded(const boost::json::value& v, K8SWatchEventDrive drive)
{
    try
    {
        VAR_FROM_JSON(std::string, generate, v.at_pointer("/metadata/generateName"));
        if (generate.empty())
        {
            return;
        }

        VAR_FROM_JSON(std::string, ip, v.at_pointer("/status/podIP"));
        if (ip.empty())
        {
            return;
        }


        VAR_FROM_JSON(std::string, name, v.at_pointer("/metadata/name"));
        if (name.empty())
        {
            return;
        }

        auto domain = name + "." + generate.substr(0, generate.size() - 1);

        if (drive == K8SWatchEventDrive::List)
        {
            StorageImp::instance().cache_[ip] = domain;
        }
        else if (drive == K8SWatchEventDrive::Watch)
        {
            StorageImp::instance().mutex_.writeLock();
            StorageImp::instance().map_[ip] = domain;
            StorageImp::instance().mutex_.unWriteLock();
        }
    }
    catch (...)
    {
    }
}

void Storage::onPodModified(const boost::json::value& v)
{
    onPodAdded(v, K8SWatchEventDrive::Watch);
}


void Storage::onPodDelete(const boost::json::value& v)
{
    try
    {
        VAR_FROM_JSON(std::string, ip, v.at_pointer("/status/podIP"));
        if (!ip.empty())
        {
            StorageImp::instance().mutex_.writeLock();
            StorageImp::instance().map_.erase(ip);
            StorageImp::instance().mutex_.unWriteLock();
        }
    }
    catch (...)
    {
    }
}

void Storage::prePodList()
{
    StorageImp::instance().cache_.clear();
}

void Storage::postPodList()
{
    {
        StorageImp::instance().mutex_.writeLock();
        std::swap(StorageImp::instance().cache_, StorageImp::instance().map_);
        StorageImp::instance().mutex_.unWriteLock();
    }
    StorageImp::instance().cache_.clear();
}

void Storage::getPodIPMap(const std::function<void(const std::unordered_map<std::string, std::string>&)>& f)
{
    StorageImp::instance().mutex_.readLock();
    f(StorageImp::instance().map_);
    StorageImp::instance().mutex_.unReadLock();
}
