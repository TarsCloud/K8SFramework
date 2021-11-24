#include "NotifyServer.h"
#include "NotifyImp.h"
#include "NotifyMsgQueue.h"

void NotifyServer::initialize()
{
    addServant<NotifyImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".NotifyObj");
    const auto& config = getConfig();
    NotifyMsgQueue::getInstance()->init(config);
}

void NotifyServer::destroyApp()
{
}