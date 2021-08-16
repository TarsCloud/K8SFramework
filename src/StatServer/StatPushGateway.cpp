#include "StatPushGateway.h"
#include "ELSPushGateway.h"
#include "ELSJSONBuilder.h"
#include <cstdio>

static void postRecord(const string &index, const string &sDate, const string &sTFlag, const StatMicMsgHead &head,
                       const StatMicMsgBody &body) {
    std::ostringstream stream;
    stream << "{";
    stream << jsonTr("source_id") << ":" << jsonTr(ServerConfig::LocalIp);
    stream << "," << jsonTr("f_timestamp") << ":" << jsonTr(TNOW);
    stream << "," << jsonTr("f_date") << ":" << jsonTr(sDate);
    stream << "," << jsonTr("f_tflag") << ":" << jsonTr(sTFlag);
    stream << "," << jsonTr("master_name") << ":" << jsonTr(head.masterName);
    stream << "," << jsonTr("slave_name") << ":" << jsonTr(head.slaveName);
    stream << "," << jsonTr("interface_name") << ":" << jsonTr(head.interfaceName);
    stream << "," << jsonTr("tars_version") << ":" << jsonTr(head.tarsVersion);
    stream << "," << jsonTr("master_ip") << ":" << jsonTr(head.masterIp);
    stream << "," << jsonTr("slave_ip") << ":" << jsonTr(head.slaveIp);
    stream << "," << jsonTr("slave_port") << ":" << jsonTr(head.slavePort);
    stream << "," << jsonTr("return_value") << ":" << jsonTr(head.returnValue);
    stream << "," << jsonTr("succ_count") << ":" << jsonTr(body.count);
    stream << "," << jsonTr("timeout_count") << ":" << jsonTr(body.timeoutCount);
    stream << "," << jsonTr("exce_count") << ":" << jsonTr(body.execCount);
    stream << "," << jsonTr("total_count") << ":" << jsonTr(body.count + body.timeoutCount + body.execCount);
    stream << "," << jsonTr("total_time") << ":" << jsonTr(body.totalRspTime);

    string strIntervCount;
    int iTemp = 0;
    for (auto &&it = body.intervalCount.begin(); it != body.intervalCount.end(); ++it) {
        if (iTemp != 0) {
            strIntervCount += ",";
        }
        strIntervCount += TC_Common::tostr(it->first);
        strIntervCount += "|";
        strIntervCount += TC_Common::tostr(it->second);
        iTemp++;
    }
    stream << "," << jsonTr("interv_count") << ":" << jsonTr(strIntervCount);
    std::size_t iAveTime = 1;
    if (body.count != 0) {
        iAveTime = body.totalRspTime / body.count;
    }
    stream << "," << jsonTr("ave_time") << ":" << jsonTr(iAveTime);
    stream << "," << jsonTr("maxrsp_time") << ":" << jsonTr(body.maxRspTime);
    stream << "," << jsonTr("minrsp_time") << ":" << jsonTr(body.minRspTime);
    stream << "}";
    ELKPushGateway::instance().postData(index, stream.str());
}

void StatPushGateway::push(const StatMicMsgHead &head, const StatMicMsgBody &body) {
    std::lock_guard<std::mutex> lockGuard(mutex);
    assert(cachePtr == cache[0] || cachePtr == cache[1]);
    cachePtr->add(head, body);
}

void StatPushGateway::sync() {

    StatHashMap *willSyncCachePtr = cachePtr;
    {
        std::lock_guard<std::mutex> lockGuard(mutex);
        assert(cachePtr == cache[0] || cachePtr == cache[1]);
        if (cachePtr == cache[0]) {
            cachePtr = cache[1];
        } else if (cachePtr == cache[1]) {
            cachePtr = cache[0];
        }
        cachePtr->clear();
    }

    std::string index = indexPre + "_" + date;
    char tflag[4 + 1] = {};
    sprintf(tflag, "%.4lu", nextSyncTFlag);
    for (auto &&item:*willSyncCachePtr) {
        StatMicMsgHead head;
        StatMicMsgBody body;
        int ret = item.get(head, body);
        if (ret < 0) {
            continue;
        }
        postRecord(index, date, tflag, head, body);
    }
}

bool StatPushGateway::isSyncTime() const {
    size_t currentSyncFlag;
    time_t t = tars::TC_TimeProvider::getInstance()->getNow();
    struct tm ptm;
    localtime_r(&t, &ptm);
    if (ptm.tm_min == 0) {
        if (ptm.tm_hour == 0) {
            currentSyncFlag = 2360;
        } else {
            currentSyncFlag = (ptm.tm_hour - 1) * 100 + 60;
        }
    } else {
        currentSyncFlag = ptm.tm_hour * 100 + ptm.tm_min;
    }
    return currentSyncFlag >= nextSyncTFlag;
}

void StatPushGateway::updateNextSyncFlag() {
    time_t t = tars::TC_TimeProvider::getInstance()->getNow();
    struct tm tm;
    localtime_r(&t, &tm);
    nextSyncTFlag = tm.tm_hour * 100 + tm.tm_min + 1;
    sprintf(date, "%.4d%.2d%.2d", tm.tm_year + 1900, tm.tm_mon + 1, tm.tm_mday);
}

void StatPushGateway::initElKPushGateway(const tars::TC_Config &config) {
    vector<string> elkNodes = config.getDomainKey("/tars/elk/nodes");
    if (elkNodes.empty()) {
        TLOGERROR("StatImp::initialize empty elk nodes " << endl);
        exit(0);
    }

    std::vector<std::tuple<std::string, int>> elkTupleNodes;
    for (auto &item : elkNodes) {
        vector<string> vOneNode = TC_Common::sepstr<string>(item, ":", true);
        if (vOneNode.size() < 2) {
            TLOGERROR("StatImp::initialize wrong elk nodes:" << item << endl);
            continue;
        }
        elkTupleNodes.emplace_back(vOneNode[0], std::stoi(vOneNode[1]));
    }

    ELKPushGateway::instance().setELKNodeAddress(elkTupleNodes);
    ELKPushGateway::instance().setFailCallback([](const std::string &errMessage) {
        string errInfo = " ERROR:" + ServerConfig::LocalIp + "_" + errMessage;
        TARS_NOTIFY_ERROR(errInfo);
        return 0;
    });
    ELKPushGateway::instance().start();
}

void StatPushGateway::initCache(const TC_Config &config) {

    TLOGDEBUG("StatServer::initHashMap begin" << endl);

    int iHashMapNum = TC_Common::strto<int>(config.get("/tars/hashmap<hashmapnum>", "1"));

    for (auto &i : cache) {
        i = new StatHashMap[iHashMapNum]();
    }

    auto iMinBlock = TC_Common::strto<int>(config.get("/tars/hashmap<minBlock>", "128"));
    auto iMaxBlock = TC_Common::strto<int>(config.get("/tars/hashmap<maxBlock>", "256"));
    auto iFactor = TC_Common::strto<float>(config.get("/tars/hashmap<factor>", "2.0"));
    auto iSize = TC_Common::toSize(config.get("/tars/hashmap<size>"), 1024 * 1024 * 64);

    for (int i = 0; i < 2; ++i) {
        for (int j = 0; j < iHashMapNum; ++j) {
            string sFileConf("/tars/hashmap<file");
            string sFileDefault("hashmap");

            sFileConf += TC_Common::tostr(i);
            sFileConf += TC_Common::tostr(j);
            sFileConf += ">";

            sFileDefault += TC_Common::tostr(i);
            sFileDefault += TC_Common::tostr(j);
            sFileDefault += ".txt";

            string sHashMapFile = ServerConfig::DataPath + "/" + config.get(sFileConf, sFileDefault);

            string sPath = TC_File::extractFilePath(sHashMapFile);

            if (!TC_File::makeDirRecursive(sPath)) {
                TLOGERROR("cannot create hashmap file " << sPath << endl);
                exit(0);
            }

            try {
                TLOGDEBUG("initDataBlockSize size: " << iMinBlock << ", " << iMaxBlock << ", " << iFactor
                                                     << ", HashMapFile:" << sHashMapFile << endl);

                cache[i][j].initDataBlockSize(iMinBlock, iMaxBlock, iFactor);

#if TARGET_PLATFORM_IOS
                cache[i][j].create(new char[iSize], iSize);
#elif TARGET_PLATFORM_WINDOWS
                cache[i][j].initStore(sHashMapFile.c_str(), iSize);
#else
                //避免一台机器上多个docker容器带来冲突
                key_t key = tars::hash<string>()(
                        string().append(ServerConfig::LocalIp).append("-").append(sHashMapFile));

                RemoteNotify::getInstance()->report(
                        "shm key:" + TC_Common::tostr(key) + ", size:" + TC_Common::tostr(iSize));

                cache[i][j].initStore(key, iSize);
#endif
                cache[i][j].setAutoErase(false);

                TLOGDEBUG("\n" << cache[i][j].desc() << endl);
            }
            catch (TC_HashMap_Exception &e) {
                RemoteNotify::getInstance()->report(string("init error: ") + e.what());

                TC_Common::msleep(100);

                TC_File::removeFile(sHashMapFile, false);
                throw runtime_error(e.what());
            }

        }
    }

    TLOGDEBUG("StatServer::initHashMap init multi hashmap end..." << endl);
}
