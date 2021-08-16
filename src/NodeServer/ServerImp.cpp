
#include "ServerImp.h"
#include "ServerManger.h"

int ServerImp::keepAlive(const tars::ServerInfo &serverInfo, tars::TarsCurrentPtr current) {
    auto serverObjectPtr = ServerManger::instance().getServer(serverInfo.application, serverInfo.serverName);

    if (serverObjectPtr != nullptr) {
        LOG->debug() << "keepAlive|" << serverInfo.adapter << "|" << serverInfo.pid << endl;
        serverObjectPtr->setPid(serverInfo.pid);
        serverObjectPtr->updateKeepAliveTime(serverInfo.adapter);
    }
    return 0;
}

int ServerImp::reportVersion(const string &app, const string &serverName, const string &version, tars::TarsCurrentPtr current) {
//    auto serverObjectPtr = ServerManger::instance().getServer(app, serverName);
//    if (serverObjectPtr != nullptr) {
//        serverObjectPtr->setPid(serverInfo.pid);
//        serverObjectPtr->updateKeepAliveTime(serverInfo.adapter);
//    }
    return 0;
}

int ServerImp::keepActiving(const tars::ServerInfo &serverInfo, tars::TarsCurrentPtr current) {
    auto serverObjectPtr = ServerManger::instance().getServer(serverInfo.application, serverInfo.serverName);
    if (serverObjectPtr != nullptr) {
        LOG->debug() << "keepActiving|" << serverInfo.adapter << "|" << serverInfo.pid << endl;
        serverObjectPtr->setPid(serverInfo.pid);
        serverObjectPtr->updateKeepActiving();
    }
    return 0;
}

tars::UInt32 ServerImp::getLatestKeepAliveTime(tars::CurrentPtr current) {
    return TNOW;
}

