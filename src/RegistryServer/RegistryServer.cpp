#include "RegistryServer.h"
#include "RegistryImp.h"
#include "QueryImp.h"
#include "K8SClient.h"
#include "K8SWatcher.h"
#include "K8SParams.h"
#include "K8SWatchCallback.h"

static bool onError(const std::error_code& ec, const std::string& msg)
{
    TLOGERROR(ec.message() << ": " << msg << std::endl);
    std::cout << ec.message() << ": " << msg << std::endl;
    return false;
}

static void createK8SContext()
{
    K8SClient::instance().start();
    K8SWatcher::instance().start();

    K8SWatcherSetting teWatchSetting("k8s.tars.io", "v1beta3", "tendpoints", K8SParams::Namespace());
    teWatchSetting.preList = K8SWatchCallback::preTEList;
    teWatchSetting.postList = K8SWatchCallback::postTEList;
    teWatchSetting.onAdded = K8SWatchCallback::onTEAdded;
    teWatchSetting.onModified = K8SWatchCallback::onTEModified;
    teWatchSetting.onDeleted = K8SWatchCallback::onTEDeleted;
    teWatchSetting.onError = onError;
    K8SWatcher::instance().addWatch(teWatchSetting);

    K8SWatcherSetting ttWatchSetting("k8s.tars.io", "v1beta3", "ttemplates", K8SParams::Namespace());
    ttWatchSetting.preList = K8SWatchCallback::preTTList;
    ttWatchSetting.postList = K8SWatchCallback::postTTList;
    ttWatchSetting.onAdded = K8SWatchCallback::onTTAdded;
    ttWatchSetting.onModified = K8SWatchCallback::onTTModified;
    ttWatchSetting.onDeleted = K8SWatchCallback::onTTDeleted;
    ttWatchSetting.onError = onError;
    K8SWatcher::instance().addWatch(ttWatchSetting);

    K8SWatcherSetting tfcWatchSetting("k8s.tars.io", "v1beta3", "tframeworkconfigs", K8SParams::Namespace());
    tfcWatchSetting.setFiledFilter("metadata.name=tars-framework");
    tfcWatchSetting.onAdded = K8SWatchCallback::onTFCAdded;
    tfcWatchSetting.onModified = K8SWatchCallback::onTFCModified;
    tfcWatchSetting.onDeleted = K8SWatchCallback::onTFCDeleted;
    tfcWatchSetting.onError = onError;
    K8SWatcher::instance().addWatch(tfcWatchSetting);
}

static void postReadinessGate()
{

    constexpr char PodNameEnv[] = "PodName";

    std::string podName = ::getenv(PodNameEnv);
    if (podName.empty())
    {
        TLOGERROR("Get Empty PodName Value" << std::endl);
        std::cout << "Get Empty PodName Value" << std::endl;
        exit(-1);
    }

    std::stringstream strStream;
    strStream.str("");
    strStream << "/api/v1/namespaces/" << K8SParams::Namespace() << "/pods/" << podName << "/status";
    const std::string setActiveUrl = strStream.str();
    strStream.str("");
    strStream << R"({"status":{"conditions":[{"type":"tars.io/active","status":"True","reason":"Active/Active"}]}})";
    const std::string setActiveBody = strStream.str();

    for (auto i = 0; i < 3; ++i)
    {
        std::this_thread::sleep_for(std::chrono::milliseconds(5));
        auto postReadinessRequest = K8SClient::instance().postRequest(K8SClientRequestMethod::StrategicMergePatch,
                setActiveUrl, setActiveBody);
        bool finish = postReadinessRequest->waitFinish(std::chrono::seconds(2));
        if (!finish)
        {
            TLOGERROR("Update Registry Server State To \"Active/Active\" Overtime" << std::endl);
            continue;
        }

        if (postReadinessRequest->state() != Done)
        {
            TLOGERROR(
                    "Update Registry Server State To \"Active/Active\" Error: " << postReadinessRequest->stateMessage()
                                                                                << std::endl);
            continue;
        }
        TLOGINFO("Update Registry Server State To \"Active/Active\" Success" << std::endl);
        return;
    }
    TLOGERROR("Update Registry Server State To \"Active/Active\" Failed" << std::endl);
    std::cout << "Update Registry Server State To \"Active/Active\" Failed" << std::endl;
    exit(-1);
}

void RegistryServer::initialize()
{

    TLOGDEBUG("RegistryServer::initialize..." << std::endl);
    std::cout << "RegistryServer::initialize..." << std::endl;

    createK8SContext();
    if (!K8SWatcher::instance().waitSync(std::chrono::seconds(30)))
    {
        TLOGERROR("RegistryServer Wait K8SWatcher Sync Overtime|30s" << std::endl);
        std::cout << "RegistryServer Wait K8SWatcher Sync Overtime|30s" << std::endl;
        exit(-1);
    }

    try
    {
        constexpr char FIXED_QUERY_SERVANT[] = "tars.tarsregistry.QueryObj";
        constexpr char FIXED_REGISTRY_SERVANT[] = "tars.tarsregistry.RegistryObj";
        addServant<QueryImp>(FIXED_QUERY_SERVANT);
        addServant<RegistryImp>(FIXED_REGISTRY_SERVANT);
    }
    catch (TC_Exception& ex)
    {
        TLOGERROR("RegistryServer Add Servant Got Exception: " << ex.what() << std::endl);
        std::cout << "RegistryServer Add Servant Got Exception: " << ex.what() << std::endl;
        exit(-1);
    }

    postReadinessGate();

    TLOGINFO("RegistryServer::initialize OK!" << std::endl);
    std::cout << "RegistryServer::initialize OK!" << std::endl;
}

void RegistryServer::destroyApp()
{
    K8SClient::instance().stop();
    K8SWatcher::instance().stop();
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    TLOGINFO("RegistryServer::destroyApp ok" << std::endl);
    std::cout << "RegistryServer::destroyApp ok" << std::endl;
}
