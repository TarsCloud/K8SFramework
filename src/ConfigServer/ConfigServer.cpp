#include "ConfigServer.h"
#include "ConfigImp.h"
#include "K8SParams.h"
#include "K8SClient.h"
#include "K8SWatcher.h"
#include "Storage.h"

static bool onError(const std::error_code& ec, const std::string& msg)
{
    TLOGERROR(ec.message() << ": " << msg << std::endl);
    std::cout << ec.message() << ": " << msg << std::endl;
    return true;
}

static void setK8SContext()
{
    K8SClient::instance().start();
    K8SWatcher::instance().start();

    K8SWatcherSetting podWatchSetting("", "v1", "pods", K8SParams::Namespace());
    podWatchSetting.setLabelFilter("tars.io/ServerApp,tars.io/ServerName");
    podWatchSetting.onAdded = Storage::onPodAdded;
    podWatchSetting.onModified = Storage::onPodModified;
    podWatchSetting.onDeleted = Storage::onPodDelete;
    podWatchSetting.preList = Storage::prePodList;
    podWatchSetting.postList = Storage::postPodList;
    podWatchSetting.onError = onError;
    K8SWatcher::instance().addWatch(podWatchSetting);
}

void ConfigServer::initialize()
{
    //滚动日志也打印毫秒
    setK8SContext();

    if (!K8SWatcher::instance().waitSync(std::chrono::seconds(20)))
    {
        TLOGERROR("ConfigServer Wait K8SWatcher Sync Overtime|20s" << std::endl);
        std::cout << "ConfigServer Wait K8SWatcher Sync Overtime|20s" << std::endl;
    }

    TarsRollLogger::getInstance()->logger()->modFlag(TC_DayLogger::HAS_MTIME);
    //增加对象
    addServant<ConfigImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".ConfigObj");

    TLOGDEBUG("ConfigServer::initialize OK!" << endl);
}

void ConfigServer::destroyApp()
{
    K8SClient::instance().stop();
    K8SWatcher::instance().stop();
    TLOGDEBUG("ConfigServer::destroy OK!" << endl);
}
