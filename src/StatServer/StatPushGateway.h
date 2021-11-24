#pragma once

#include "servant/Application.h"
#include "servant/StatReport.h"
#include "StatHashMap.h"
#include "util/tc_timer.h"
#include "ESHelper.h"
#include <mutex>

class StatPushGateway
{
public:
	static StatPushGateway& instance()
	{
		static StatPushGateway gateway;
		return gateway;
	}

	void init(const tars::TC_Config& config);

	void push(const tars::StatMicMsgHead& head, const tars::StatMicMsgBody& body);

	void updateNextSyncFlag();

	void start();

private:

	StatPushGateway() = default;

	void initCache(const tars::TC_Config& config);

	void sync();

	bool isSyncTime() const;

private:
	std::mutex mutex;
	StatHashMap* cachePtr{};
	std::array<StatHashMap*, 2> cache{};
	std::string indexPre{};
	std::thread thread{};
	char date[9]{};
	std::size_t nextSyncTFlag{};
	TC_Timer timer_{};
};


