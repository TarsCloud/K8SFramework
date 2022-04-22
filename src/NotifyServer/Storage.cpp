
#include "Storage.h"
#include "util/tc_thread_rwlock.h"
#include <rapidjson/pointer.h>
#include <string>

static inline std::string JP2S(const rapidjson::Value* p)
{
	assert(p != nullptr && p->IsString());
	return { p->GetString(), p->GetStringLength() };
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
	std::unordered_map <std::string, std::string> map_;
	std::unordered_map <std::string, std::string> cache_;
};

void Storage::onPodAdded(const rapidjson::Value& v, K8SWatchEventDrive drive)
{
	std::string sPodIP{};
	auto pPodIP = rapidjson::GetValueByPointer(v, "/status/podIP");

	if (pPodIP == nullptr)
	{
		return;
	}

	assert(pPodIP->IsString());
	sPodIP = JP2S(pPodIP);

	if (sPodIP.empty())
	{
		return;
	}

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

	if (drive == K8SWatchEventDrive::List)
	{
		StorageImp::instance().cache_[sPodIP] = sDomain;
	}
	else if (drive == K8SWatchEventDrive::Watch)
	{
		StorageImp::instance().mutex_.writeLock();
		StorageImp::instance().map_[sPodIP] = sDomain;
		StorageImp::instance().mutex_.unWriteLock();
	}
}

void Storage::onPodModified(const rapidjson::Value& v)
{
	onPodAdded(v, K8SWatchEventDrive::Watch);
}


void Storage::onPodDelete(const rapidjson::Value& v)
{
	std::string sPodIP{};
	auto pPodIP = rapidjson::GetValueByPointer(v, "/status/podIP");
	if (pPodIP == nullptr)
	{
		return;
	}

	assert(pPodIP->IsString());
	sPodIP = JP2S(pPodIP);
	if (sPodIP.empty())
	{
		return;
	}

	StorageImp::instance().mutex_.writeLock();
	StorageImp::instance().map_.erase(sPodIP);
	StorageImp::instance().mutex_.unWriteLock();
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

void Storage::getPodIPMap(const std::function<void(const std::unordered_map <std::string, std::string>&)>& f)
{
	StorageImp::instance().mutex_.readLock();
	f(StorageImp::instance().map_);
	StorageImp::instance().mutex_.unReadLock();
}
