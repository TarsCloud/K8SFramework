
#include "Storage.h"
#include "util/tc_thread_rwlock.h"
#include "TCodec.h"
#include <string>

static int extractPodSeq(const std::string& sPodName, const std::string& sGenerateName)
{
    auto pos = sGenerateName.size();
    if (pos == sPodName.size())
    {
        return -1;
    }

    if (sPodName.compare(0, pos, sGenerateName) != 0)
    {
        return -1;
    }

    auto d = 0;
    for (; pos != sPodName.size(); ++pos)
    {
        auto c = sPodName[pos];
        if (c >= '0' && c <= '9')
        {
            d = d * 10 + c - '0';
            continue;
        }
        return -1;
    }
    return pos == sPodName.size() ? d : -1;
}

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
    std::unordered_map<std::string, int> seqMap_;
    std::unordered_map<std::string, int> cacheSeqMap_;
};

void Storage::onPodAdded(const boost::json::value& v, K8SWatchEventDrive drive)
{
    try
    {
        VAR_FROM_JSON(std::string, ip, v.at_pointer("/status/podIP"));
        if (!ip.empty())
        {
            return;
        }

        VAR_FROM_JSON(std::string, generate, v.at_pointer("/metadata/generateName"));
        if (generate.empty())
        {
            return;
        }

        VAR_FROM_JSON(std::string, name, v.at_pointer("/metadata/name"));
        if (name.empty())
        {
            return;
        }

        int seq = extractPodSeq(name, generate);
        if (seq == -1)
        {
            return;
        }

        if (drive == K8SWatchEventDrive::List)
        {
            StorageImp::instance().cacheSeqMap_[ip] = seq;
        }
        else if (drive == K8SWatchEventDrive::Watch)
        {
            StorageImp::instance().mutex_.writeLock();
            StorageImp::instance().seqMap_[ip] = seq;
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
            StorageImp::instance().seqMap_.erase(ip);
            StorageImp::instance().mutex_.unWriteLock();
        }
    }
    catch (...)
    {
    }
}

void Storage::prePodList()
{
    StorageImp::instance().cacheSeqMap_.clear();
}

void Storage::postPodList()
{
    {
        StorageImp::instance().mutex_.writeLock();
        std::swap(StorageImp::instance().cacheSeqMap_, StorageImp::instance().seqMap_);
        StorageImp::instance().mutex_.unWriteLock();
    }
    StorageImp::instance().cacheSeqMap_.clear();
}

void Storage::getSeqMap(const std::function<void(const std::unordered_map<std::string, int>&)>& f)
{
    StorageImp::instance().mutex_.readLock();
    f(StorageImp::instance().seqMap_);
    StorageImp::instance().mutex_.unReadLock();
}
