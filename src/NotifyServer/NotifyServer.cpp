#include "NotifyServer.h"
#include "NotifyImp.h"
#include "NotifyMsgQueue.h"
#include <K8SParams.h>
#include <K8SWatcher.h>
#include "Storage.h"

static bool onError(const std::error_code& ec, const std::string& msg)
{
    TLOGERROR(ec.message() << ": " << msg << std::endl);
    std::cout << ec.message() << ": " << msg << std::endl;
    return true;
}

static void setK8SContext()
{
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

void NotifyServer::initialize()
{
    setK8SContext();
    addServant<NotifyImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".NotifyObj");
    const auto& config = getConfig();
    NotifyMsgQueue::getInstance()->init(config);
}

void NotifyServer::destroyApp()
{
}