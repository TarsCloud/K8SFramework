#pragma once

#include <mutex>
#include <servant/Application.h>
#include "servant/StatReport.h"
#include "PropertyHashMap.h"

class PropertyPushGateway {
public:
    static PropertyPushGateway &instance() {
        static PropertyPushGateway gateway;
        return gateway;
    }

    void init(const tars::TC_Config &config) {
        _indexPre = config.get("/tars/elk<indexPre>", "property_");
        initCache(config);
        _cachePtr = _cache[0];
        initElKPushGateway(config);
    }

    void push(const tars::StatPropMsgHead &head, const tars::StatPropMsgBody &body);

    void updateNextSyncFlag();

    inline void start() {
        _thread = std::thread([this] {
            while (true) {
                if (isSyncTime()) {
                    sync();
                    updateNextSyncFlag();
                }
                usleep(7 * 1000 * 1000);
            }
        });
        _thread.detach();
    }

private:

    PropertyPushGateway() {
        updateNextSyncFlag();
    }

    void initElKPushGateway(const tars::TC_Config &config);

    void initCache(const tars::TC_Config &config);

    void sync();

    bool isSyncTime() const;

private:
    std::mutex _mutex;
    PropertyHashMap *_cachePtr = NULL;
    PropertyHashMap *_cache[2];
    std::string _indexPre{};
    std::thread _thread{};
    char _date[9]{};
    std::size_t _nextSyncTFlag{};
};


