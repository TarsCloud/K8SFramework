#pragma once

#include <mutex>
#include <servant/Application.h>
#include "servant/StatReport.h"
#include "StatHashMap.h"

class StatPushGateway {
public:
    static StatPushGateway &instance() {
        static StatPushGateway gateway;
        return gateway;
    }

    void init(const tars::TC_Config &config) {
        indexPre = config.get("/tars/elk<indexPre>", "stat_");
        initCache(config);
        cachePtr = cache[0];
        initElKPushGateway(config);
    }

    void push(const tars::StatMicMsgHead &head, const tars::StatMicMsgBody &body);

    void updateNextSyncFlag();

    inline void start() {
        thread = std::thread([this] {
            while (true) {
                if (isSyncTime()) {
                    sync();
                    updateNextSyncFlag();
                }
                usleep(7 * 1000 * 1000);
            }
        });
        thread.detach();
    }

private:

    StatPushGateway() {
        updateNextSyncFlag();
    }

    void initElKPushGateway(const tars::TC_Config &config);

    void initCache(const tars::TC_Config &config);

    void sync();

    bool isSyncTime() const;

private:
    std::mutex mutex;
    StatHashMap *cachePtr{};
    std::array<StatHashMap *, 2> cache{};
    std::string indexPre{};
    std::thread thread{};
    char date[9]{};
    std::size_t nextSyncTFlag{};
};


