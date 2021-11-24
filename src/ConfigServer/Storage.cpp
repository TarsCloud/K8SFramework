
#include "Storage.h"
#include "util/tc_thread_rwlock.h"
#include <rapidjson/pointer.h>
#include <string>

static inline std::string JP2S(const rapidjson::Value* p)
{
    assert(p != nullptr && p->IsString());
    return { p->GetString(), p->GetStringLength() };
}

static int extractPodSeq(const std::string& sPodName, const std::string& sGenerateName)
{
    try
    {
        assert(sPodName.size() > sGenerateName.size());
        auto sPodSeq = sPodName.substr(sGenerateName.size());
        return std::stoi(sPodSeq, nullptr, 10);
    }
    catch (std::exception& exception)
    {
        return -1;
    }
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

void Storage::onPodAdded(const rapidjson::Value& v, K8SWatchEventDrive drive)
{
    auto pGenerateName = rapidjson::GetValueByPointer(v, "/metadata/generateName");
    if (pGenerateName == nullptr)
    {
        return;
    }
    assert(pGenerateName->IsString());
    auto sGenerateName = JP2S(pGenerateName);

    auto pPodName = rapidjson::GetValueByPointer(v, "/metadata/name");
    assert(pPodName != nullptr && pPodName->IsString());
    std::string sPodName = JP2S(pPodName);

    auto sDomain = sPodName + "." + sGenerateName.substr(0, sGenerateName.size() - 1);

    std::string sPodIP{};
    auto pPodIP = rapidjson::GetValueByPointer(v, "/status/podIP");
    if (pPodIP != nullptr)
    {
        assert(pPodIP->IsString());
        sPodIP = JP2S(pPodIP);
    }

    int iPodSeq = extractPodSeq(sPodName, sGenerateName);
    if (iPodSeq == -1)
    {
        return;
    }

    if (drive == K8SWatchEventDrive::List)
    {
        StorageImp::instance().cacheSeqMap_[sPodName] = iPodSeq;
        StorageImp::instance().cacheSeqMap_[sDomain] = iPodSeq;
        if (!sPodIP.empty())
        {
            StorageImp::instance().cacheSeqMap_[sPodIP] = iPodSeq;
        }
    }
    else if (drive == K8SWatchEventDrive::Watch)
    {
        StorageImp::instance().mutex_.writeLock();
        StorageImp::instance().seqMap_[sPodName] = iPodSeq;
        StorageImp::instance().seqMap_[sDomain] = iPodSeq;
        if (!sPodIP.empty())
        {
            StorageImp::instance().seqMap_[sPodIP] = iPodSeq;
        }
        StorageImp::instance().mutex_.unWriteLock();
    }
}

void Storage::onPodModified(const rapidjson::Value& v)
{
    onPodAdded(v, K8SWatchEventDrive::Watch);
}

void Storage::onPodDelete(const rapidjson::Value& v)
{
    auto pGenerateName = rapidjson::GetValueByPointer(v, "/metadata/generateName");
    if (pGenerateName == nullptr)
    {
        return;
    }
    assert(pGenerateName->IsString());
    auto sGenerateName = JP2S(pGenerateName);

    auto pPodName = rapidjson::GetValueByPointer(v, "/metadata/name");
    assert(pPodName != nullptr && pPodName->IsString());
    std::string sPodName = JP2S(pPodName);

    auto sDomain = sPodName + "." + sGenerateName.substr(0, sGenerateName.size() - 1);

    std::string sPodIP{};
    auto pPodIP = rapidjson::GetValueByPointer(v, "/status/podIP");
    if (pPodIP != nullptr)
    {
        assert(pPodIP->IsString());
        sPodIP = JP2S(pPodIP);
    }

    StorageImp::instance().mutex_.writeLock();
    StorageImp::instance().seqMap_.erase(sPodName);
    StorageImp::instance().seqMap_.erase(sDomain);
    if (!sPodIP.empty())
    {
        StorageImp::instance().seqMap_.erase(sPodIP);
    }
    StorageImp::instance().mutex_.unWriteLock();
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
