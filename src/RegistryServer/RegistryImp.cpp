#include "RegistryImp.h"
#include "RegistryServer.h"
#include "util/tc_mysql.h"
#include "ServerInfoInterface.h"
#include "K8SClient.h"
#include "K8SParams.h"

void RegistryImp::initialize() {
    LOG->debug() << "RegistryImp init ok." << endl;
}

void RegistryImp::updateServerState(const std::string &podName, const std::string &settingState, const std::string &presentState, CurrentPtr current) {
    std::stringstream strStream;
    strStream.str("");
    strStream << "/api/v1/namespaces/" << K8SParams::instance().bindNamespace() << "/pods/" << podName << "/status";
    const std::string patchUrl = strStream.str();

    strStream.str("");
    strStream << R"({"status":{"conditions":[{"type":"tars.io/active")" << ","
              << R"("status":")" << ((settingState == "Active" && presentState == "Active") ? "True" : "False") << R"(",)"
              << R"("reason":")" << settingState << "/" << presentState << R"("}]}})";

    const std::string patchBody = strStream.str();
    constexpr int MAX_RETRIES_TIMES = 5;
    for (auto i = 0; i < MAX_RETRIES_TIMES; ++i) {

        auto patchRequest = K8SClient::instance().postRequest(K8SClientRequestMethod::StrategicMergePatch, patchUrl, patchBody);

        bool finish = patchRequest->waitFinish(std::chrono::seconds(3));
        if (!finish) {
            LOG->error() << "Update Server State Overtime" << endl;
            continue;
        }

        if (patchRequest->state() != Done) {
            LOG->error() << "Update Server State Error: " << string(patchRequest->responseBody(),patchRequest->responseSize())<< endl;
            continue;
        }

        if (patchRequest->responseCode() != HTTP_STATUS_OK) {
            LOG->error() << "Update Server State Error: " << string(patchRequest->responseBody(),patchRequest->responseSize())<< endl;
            continue;
        }

        return;
    }
    LOG->error()<<"Update Server State Error, "<<MAX_RETRIES_TIMES<<std::endl;
}

Int32 RegistryImp::getServerDescriptor(const std::string &serverApp, const std::string &serverName, ServerDescriptor &serverDescriptor, CurrentPtr current) {
    return ServerInfoInterface::instance().getServerDescriptor(serverApp, serverName, serverDescriptor);
}
