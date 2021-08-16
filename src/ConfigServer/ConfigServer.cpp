#include "ConfigServer.h"
#include "ConfigImp.h"
#include "K8SParams.h"
#include "K8SClient.h"
#include "K8SWatcher.h"
#include "ConfigInfoInterface.h"

void ConfigServer::initialize() {
    //滚动日志也打印毫秒
    TarsRollLogger::getInstance()->logger()->modFlag(TC_DayLogger::HAS_MTIME);
    //增加对象
    addServant<ConfigImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".ConfigObj");

    K8SParams::instance().init();

    std::string podsWatchUrl =
            std::string("/api/v1/namespaces/").append(K8SParams::instance().bindNamespace()).append("/pods");
    K8SWatcher::instance().postWatch(podsWatchUrl, [](K8SWatchEvent type, const rapidjson::Value &value) {
        switch (type) {
            case K8SWatchEventAdded: {
                ConfigInfoInterface::instance().onPodAdd(value);
            }
                break;
            case K8SWatchEventDeleted: {
                ConfigInfoInterface::instance().onPodDelete(value);
            }
                break;
            case K8SWatchEventUpdate: {
                ConfigInfoInterface::instance().onPodUpdate(value);
            }
                break;
            default:
                break;
        }
    });
    K8SClient::instance().start();
    K8SWatcher::instance().start();
    TLOGDEBUG("ConfigServer::initialize OK!" << endl);
}

void ConfigServer::destroyApp() {
    TLOGDEBUG("ConfigServer::destroy OK!" << endl);
}
