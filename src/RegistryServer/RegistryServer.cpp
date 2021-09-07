#include "RegistryServer.h"
#include "RegistryImp.h"
#include "QueryImp.h"
#include "ServerInfoInterface.h"
#include "K8SClient.h"
#include "K8SWatcher.h"
#include "K8SParams.h"
#include "util/tc_timer.h"

void RegistryServer::initialize() {

    LOG->debug() << "RegistryServer::initialize..." << endl;

    K8SParams::instance().init();

    std::string tendpointsWatchUrl =
            std::string("/apis/k8s.tars.io/v1beta1/namespaces/") + K8SParams::instance().bindNamespace() +
            "/tendpoints";
    K8SWatcher::instance().postWatch(tendpointsWatchUrl, handleEndpointsEvent);

    std::string ttemplatesWatchUrl =
            std::string("/apis/k8s.tars.io/v1beta1/namespaces/") + K8SParams::instance().bindNamespace() +
            "/ttemplates";
    K8SWatcher::instance().postWatch(ttemplatesWatchUrl, handleTemplateEvent);

    K8SClient::instance().start();
    K8SWatcher::instance().start();

    constexpr char PodNameEnv[] = "PodName";

    std::string podName = ::getenv(PodNameEnv);
    if (podName.empty()) {
        LOG->error() << "Get Empty PodName Value ,Program Will Exit " << endl;
        cerr << "Get Empty PodName Value ,Program Will Exit " << endl;
    }

    try {
        constexpr char FIXED_QUERY_SERVANT[] = "tars.tarsregistry.QueryObj";
        constexpr char FIXED_REGISTRY_SERVANT[] = "tars.tarsregistry.RegistryObj";
        addServant<QueryImp>(FIXED_QUERY_SERVANT);
        addServant<RegistryImp>(FIXED_REGISTRY_SERVANT);
    } catch (TC_Exception &ex) {
        LOG->error() << "RegistryServer initialize exception:" << ex.what() << endl;
        cerr << "RegistryServer initialize exception:" << ex.what() << endl;
        LOG->flush();
        exit(-1);
    }

    std::stringstream strStream;
    strStream.str("");
    strStream << "/api/v1/namespaces/" << K8SParams::instance().bindNamespace() << "/pods/" << podName
              << "/status";
    const std::string setActiveUrl = strStream.str();
    strStream.str("");
    strStream << R"({"status":{"conditions":[{"type":"tars.io/active","status":"True","reason":"Active/Active"}]}})";
    const std::string setActiveBody = strStream.str();

    auto setActiveTask = K8SClient::instance().postRequest(K8SClientRequestMethod::StrategicMergePatch, setActiveUrl,
                                                           setActiveBody);
    bool finish = setActiveTask->waitFinish(std::chrono::seconds(1));
    if (!finish) {
        LOG->error() << "Set Registry Server State To \"Active/Active\" Overtime" << std::endl;
        exit(-1);
    }

    if (setActiveTask->state() != Done) {
        LOG->error() << "Set Registry Server State To \"Active/Active\" Error: " << setActiveTask->stateMessage()
                     << std::endl;
        exit(-1);
    }

    upchainThread_ = std::thread([]() {
        while (true) {
            ServerInfoInterface::instance().loadUpChainConf();
            usleep(3 * 1000 * 1000);
        }
    });

    LOG->debug() << "Set Registry Server State To \"Active/Active\" Success" << std::endl;
    LOG->debug() << "RegistryServer::initialize OK!" << std::endl;
}

void RegistryServer::destroyApp() {
    LOG->error() << "RegistryServer::destroyApp ok" << std::endl;
}
