#pragma once

#include "K8SWatcher.h"
#include "servant/EndpointF.h"
#include "util/tc_thread_rwlock.h"
#include <string>
#include <map>
#include <vector>
#include <set>

struct TTemplate
{
	std::string name;
	std::string parent;
	std::string content;
};

struct TAdapter
{
	bool isTcp{};
	bool isTars{};
	int port{};
	int32_t thread{};
	int32_t connection{};
	int32_t timeout{};
	int32_t capacity{};
	std::string name;
};

struct UpChain
{
	std::map<std::string, std::vector<tars::EndpointF>> customs;
	std::vector<tars::EndpointF> defaults;
};

struct TEndpoint
{
	int asyncThread{};
	std::string profileContent;
	std::string templateName;
	std::vector<std::shared_ptr<TAdapter>> tadapters;
	std::set<std::string> activatedPods;
	std::set<std::string> inActivatedPods;
};

class Storage
{
public:
	static Storage& instance()
	{
		static Storage store;
		return store;
	}

	inline void preListTemplate()
	{
		cacheTemplates_.clear();
	}

	inline void postListTemplate()
	{
		{
			templateMutex_.writeLock();
			std::swap(templates_, cacheTemplates_);
			templateMutex_.unWriteLock();
		}
		cacheTemplates_.clear();
	}

	inline void updateTemplate(const std::string& name, const std::shared_ptr<TTemplate>& t, K8SWatchEventDrive driver)
	{
		if (driver == K8SWatchEventDrive::List)
		{
			cacheTemplates_[t->name] = t;
			return;
		}
		templateMutex_.writeLock();
		templates_[t->name] = t;
		templateMutex_.unWriteLock();
	}

	inline void deleteTemplate(const std::string& name)
	{
		templateMutex_.writeLock();
		templates_.erase(name);
		templateMutex_.unWriteLock();
	}

	inline void preListEndpoint()
	{
		cacheEndpoints_.clear();
	}

	inline void postListEndpoint()
	{
		{
			endpointMutex_.writeLock();
			std::swap(endpoints_, cacheEndpoints_);
			endpointMutex_.unWriteLock();
		}
		cacheEndpoints_.clear();
	}

	inline void updateEndpoint(const std::string& name, const std::shared_ptr<TEndpoint>& t, K8SWatchEventDrive driver)
	{
		if (driver == K8SWatchEventDrive::List)
		{
			cacheEndpoints_[name] = t;
			return;
		}
		endpointMutex_.writeLock();
		endpoints_[name] = t;
		endpointMutex_.unWriteLock();
	}

	inline void deleteTEndpoint(const std::string& name)
	{
		endpointMutex_.writeLock();
		endpoints_.erase(name);
		endpointMutex_.unWriteLock();
	}

	inline void updateUpChain(const std::shared_ptr<UpChain>& upChain)
	{
		upChainMutex_.writeLock();
		upChain_ = upChain;
		upChainMutex_.unWriteLock();
	}

	inline void getTEndpoints(const std::function<void(const std::map<std::string, std::shared_ptr<TEndpoint>>&)>& f)
	{
		endpointMutex_.readLock();
		f(endpoints_);
		endpointMutex_.unReadLock();
	}

	inline void getTTemplates(const std::function<void(const std::map<std::string, std::shared_ptr<TTemplate>>&)>& f)
	{
		templateMutex_.readLock();
		f(templates_);
		templateMutex_.unReadLock();
	}

	inline void getUnChain(const std::function<void(std::shared_ptr<UpChain>&)>& f)
	{
		upChainMutex_.readLock();
		f(upChain_);
		upChainMutex_.unReadLock();
	}


	~Storage() = default;

	Storage(const Storage&) = delete;

private:
	Storage() = default;

private:
	tars::TC_ThreadRWLocker endpointMutex_;;
	std::map<std::string, std::shared_ptr<TEndpoint>> endpoints_;
	std::map<std::string, std::shared_ptr<TEndpoint>> cacheEndpoints_;

	tars::TC_ThreadRWLocker templateMutex_;
	std::map<std::string, std::shared_ptr<TTemplate>> templates_;
	std::map<std::string, std::shared_ptr<TTemplate>> cacheTemplates_;

	tars::TC_ThreadRWLocker upChainMutex_;
	std::shared_ptr<UpChain> upChain_;
};
