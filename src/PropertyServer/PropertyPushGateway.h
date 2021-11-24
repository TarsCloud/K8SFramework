#pragma once

#include <mutex>
#include <servant/Application.h>
#include "servant/StatReport.h"
#include "PropertyHashMap.h"
#include "util/tc_timer.h"

class PropertyPushGateway
{
public:
	static PropertyPushGateway& instance()
	{
		static PropertyPushGateway gateway;
		return gateway;
	}

	void init(const tars::TC_Config& config);

	void push(const tars::StatPropMsgHead& head, const tars::StatPropMsgBody& body);

	void updateNextSyncFlag();

	void start();

private:

	PropertyPushGateway() = default;

	void initCache(const tars::TC_Config& config);

	void sync();

	bool isSyncTime() const;

private:
	std::mutex _mutex;
	PropertyHashMap* _cachePtr = nullptr;
	PropertyHashMap* _cache[2]{};
	std::string _indexPre{};
	char _date[9]{};
	std::size_t _nextSyncTFlag{};
	TC_Timer timer_;
};


