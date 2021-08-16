#include "NodeImp.h"
#include "NodeServer.h"
#include "ServerManger.h"
#include "util/tc_timeprovider.h"
#include "TimerTaskQueue.h"
#include "util.h"

void NodeImp::initialize() {
}

int NodeImp::startServer(const string &application, const string &serverName, string &result, TarsCurrentPtr current) {
//    TLOG_DEBUG("startServer:" << application << "," << serverName << endl);
    auto pServerObjectPtr = ServerManger::instance().getServer(application, serverName);
    if (pServerObjectPtr == nullptr) {
        result += FILE_FUN_STR + "server not exist";
        return -1;
    }
    return pServerObjectPtr->startServer(result);
}

int NodeImp::stopServer(const string &application, const string &serverName, string &result, TarsCurrentPtr current) {
//    TLOG_DEBUG("stopServer:" << application << "," << serverName << endl);
    auto pServerObjectPtr = ServerManger::instance().getServer(application, serverName);
    if (pServerObjectPtr == nullptr) {
        result += FILE_FUN_STR + "server not exist";
        return -1;
    }
    return pServerObjectPtr->stopServer(result);
}

int NodeImp::restartServer(const std::string &application, const std::string &serverName, std::string &result, tars::TarsCurrentPtr current) {
//    TLOG_DEBUG("restartServer:" << application << "," << serverName << endl);
    auto pServerObjectPtr = ServerManger::instance().getServer(application, serverName);
    if (pServerObjectPtr == nullptr) {
        result += FILE_FUN_STR + "server not exist";
        return -1;
    }
    return pServerObjectPtr->restartServer(result);
}

int NodeImp::notifyServer(const string &application, const string &serverName, const string &sMsg, string &result, TarsCurrentPtr current) {
//    TLOG_DEBUG("notifyServer:" << application << "," << serverName << endl);
    auto pServerObjectPtr = ServerManger::instance().getServer(application, serverName);
    if (pServerObjectPtr == nullptr) {
        result += FILE_FUN_STR + "server not exist";
        return -1;
    }
    return pServerObjectPtr->notifyServer(sMsg, result);
}

int NodeImp::addFile(const string &application, const string &serverName, const string &file, string &result, TarsCurrentPtr current) {
//    TLOG_DEBUG("addFile:" << application << "." << serverName << ":" << file << endl);
    if (file.empty()) {
        result = FILE_FUN_STR + "file is empty" + file;
        LOG->debug() << result << endl;
        return -1;
    }

    auto pServerObjectPtr = ServerManger::instance().getServer(application, serverName);

    if (pServerObjectPtr == nullptr) {
        result += FILE_FUN_STR + "server not exist";
        return -1;
    }

    return pServerObjectPtr->addFile(file, result);
}





