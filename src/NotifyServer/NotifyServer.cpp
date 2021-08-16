#include "NotifyServer.h"
#include "NotifyImp.h"
#include "NotifyMsgQueue.h"

void NotifyServer::initialize() {
    addServant<NotifyImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".NotifyObj");
    NotifyMsgQueue::getInstance()->init();
}

void NotifyServer::destroyApp() {
    // cout << "NotifyServer::destroyApp ok" << endl;
}
