#include "KEventServer.h"
#include "K8SWatchCallback.h"
#include "K8SClient.h"
#include "K8SWatcher.h"
#include "K8SParams.h"
#include "ESHelper.h"

static bool onError(const std::error_code& ec, const std::string& msg)
{
    TLOGERROR(ec.message() << ": " << msg << std::endl);
    std::cout << ec.message() << ": " << msg << std::endl;
    return true;
}

static void createK8SContext()
{
    K8SClient::instance().start();
    K8SWatcher::instance().start();

    K8SWatcherSetting eventsWatchSetting("", "v1", "events", K8SParams::Namespace());
    eventsWatchSetting.onAdded = K8SWatchCallback::onEventsAdded;
    eventsWatchSetting.onModified = K8SWatchCallback::onEventsModified;
    eventsWatchSetting.onDeleted = K8SWatchCallback::onEventsDeleted;
    eventsWatchSetting.onError = onError;
    K8SWatcher::instance().addWatch(eventsWatchSetting);

    if (K8SParams::Namespace() != "default")
    {
        K8SWatcherSetting clusterWatchSetting("", "v1", "events", "default");
        clusterWatchSetting.onAdded = K8SWatchCallback::onEventsAddedWithFilter;
        clusterWatchSetting.onModified = K8SWatchCallback::onEventsModifiedWithFilter;
        clusterWatchSetting.onDeleted = K8SWatchCallback::onEventsDeleted;
        clusterWatchSetting.onError = onError;
        K8SWatcher::instance().addWatch(clusterWatchSetting);
    }
}

void KEventServer::initialize()
{
    TLOGDEBUG("KEventServer::initialize..." << std::endl);
    std::cout << "KEventServer::initialize..." << std::endl;

    const auto& config = getConfig();

    auto index = config.get("/tars/elk/index<kevent>");
    if (index.empty())
    {
        auto message = std::string("get empty index value");
        TLOGERROR(message << std::endl);
        throw std::runtime_error(message);
    }

    K8SWatchCallback::setESIndex(index);

    auto age = config.get("/tars/elk/age<kevent>", "3d");
    const auto& _template = index;
    const auto& pattern = index;
    const auto& policy = index;

    ESHelper::setAddressByTConfig(config);
    ESHelper::createESPolicy(policy, age);
    ESHelper::createESDataStreamTemplate(_template, pattern, policy);

    createK8SContext();
    if (!K8SWatcher::instance().waitSync(std::chrono::seconds(30)))
    {
        TLOGERROR("KEventServer Wait K8SWatcher Sync Overtime|30s" << std::endl);
        std::cout << "KEventServer Wait K8SWatcher Sync Overtime|30s" << std::endl;
        exit(-1);
    }

    TLOGINFO("KEventServer::initialize OK!" << std::endl);
    std::cout << "KEventServer::initialize OK!" << std::endl;
}

void KEventServer::destroyApp()
{
    K8SClient::instance().stop();
    K8SWatcher::instance().stop();
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    TLOGINFO("KEventServer::destroyApp ok" << std::endl);
    std::cout << "KEventServer::destroyApp ok" << std::endl;
}
