
#include "ConfigInfoInterface.h"
#include "K8SClient.h"
#include "K8SParams.h"
#include <rapidjson/pointer.h>
#include <set>

//多配置文件的分割符
constexpr char MultiConfigSeparator[] = "\r\n\r\n";

static int extractPodSeq(const std::string &sPodName, const std::string &sGenerateName) {
    try {
        auto sPodSeq = sPodName.substr(sGenerateName.size());
        return std::stoi(sPodSeq, nullptr, 10);
    } catch (std::exception &exception) {
        return -1;
    }
}

static inline std::string SFromP(const rapidjson::Value *p) {
    assert(p != nullptr);
    return {p->GetString(), p->GetStringLength()};
}

void ConfigInfoInterface::onPodAdd(const rapidjson::Value &pDocument) {
    auto pGenerateName = rapidjson::GetValueByPointer(pDocument, "/metadata/generateName");
    if (pGenerateName == nullptr) { return; }
    assert(pGenerateName->IsString());
    std::string sGenerateName = SFromP(pGenerateName);

    auto pPodIP = rapidjson::GetValueByPointer(pDocument, "/status/podIP");
    if (pPodIP == nullptr) { return; }
    assert(pPodIP->IsString());
    std::string podIP = SFromP(pPodIP);

    auto pPodName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(pPodName != nullptr && pPodName->IsString());
    std::string sPodName = SFromP(pPodName);

    int iPodSeq = extractPodSeq(sPodName, sGenerateName);

    std::lock_guard<std::mutex> lockGuard(mutex_);
    ipPodSeqMap_[podIP] = iPodSeq;
}

void ConfigInfoInterface::onPodUpdate(const rapidjson::Value &pDocument) {
    return onPodAdd(pDocument);
}

void ConfigInfoInterface::onPodDelete(const rapidjson::Value &pDocument) {
    auto pPodIP = rapidjson::GetValueByPointer(pDocument, "/status/podIP");
    if (pPodIP == nullptr) { return; }
    assert(pPodIP->IsString());
    std::string sPodIP = SFromP(pPodIP);

    std::lock_guard<std::mutex> lockGuard(mutex_);
    ipPodSeqMap_.erase(sPodIP);
}

ConfigInfoInterface::GetConfigResult
ConfigInfoInterface::loadConfig(const std::string &sSeverApp, const std::string &sSeverName,
                                const std::string &sConfigName, const std::string &sClientIP,
                                std::string &sConfigContent, std::string &sErrorInfo) {
    if (sSeverName.empty()) {
        return loadAppConfig(sSeverApp, sConfigName, sConfigContent, sErrorInfo);
    }
    return loadServerConfig(sSeverApp, sSeverName, sConfigName, sClientIP, sConfigContent, sErrorInfo);
}

ConfigInfoInterface::GetConfigResult
ConfigInfoInterface::loadAppConfig(const std::string &sServerApp, const std::string &sConfigName,
                                   std::string &sConfigContent, std::string &sErrorInfo) {
    sConfigContent.clear();
    std::ostringstream stream;
    stream << "/apis/k8s.tars.io/v1alpha1/namespaces/" << K8SParams::instance().bindNamespace()
           << "/tconfigs?labelSelector="
           << "tars.io/ServerApp=" << sServerApp
           << ",tars.io/ServerName="
           << ",tars.io/ConfigName=" << sConfigName
           << ",tars.io/Activated=true"
           << ",!tars.io/Deleting";

    bool existDeactivateMasterConfig{false};
    constexpr int MAX_RETRIES_LOAD_TIMES = 3;

    for (int i = 0; i < MAX_RETRIES_LOAD_TIMES; ++i) {
        usleep(i * 30 * 1000);
        auto k8sClientRequest = K8SClient::instance().postRequest(K8SClientRequestMethod::Get, stream.str(), "");
        bool bTaskFinish = k8sClientRequest->waitFinish(std::chrono::seconds(2));

        if (!bTaskFinish) {
            sErrorInfo.append(k8sClientRequest->stateMessage());
            continue;
        }

        if (k8sClientRequest->state() != Done) {
            sErrorInfo.append(k8sClientRequest->stateMessage());
            continue;
        }

        const auto &responseJson = k8sClientRequest->responseJson();
        auto pItem = rapidjson::GetValueByPointer(responseJson, "/items");
        if (pItem == nullptr) {
            sErrorInfo.append(k8sClientRequest->responseBody(), k8sClientRequest->responseSize());
            continue;
        }

        assert(pItem->IsArray());
        for (auto &&config:pItem->GetArray()) {
            auto pConfigContent = rapidjson::GetValueByPointer(config, "/configContent");
            assert(pConfigContent != nullptr);

            auto pConfigDeactivate = rapidjson::GetValueByPointer(config, "/metadata/labels/Deactivate");
            if (pConfigDeactivate == nullptr) {
                sConfigContent = SFromP(pConfigContent);
                return ConfigInfoInterface::Success;
            }

            assert(pConfigDeactivate != nullptr);
            existDeactivateMasterConfig = true;
        }
        if (existDeactivateMasterConfig) {
            continue;
        }
        sErrorInfo = "config not existed or not activated";
        return ConfigInfoInterface::ConfigError;
    }
    return ConfigInfoInterface::K8SError;
}

ConfigInfoInterface::GetConfigResult
ConfigInfoInterface::loadServerConfig(const std::string &sServerApp, const std::string &sSeverName,
                                      const std::string &sConfigName, const std::string &sClientIP,
                                      std::string &sConfigContent, std::string &sErrorInfo) {
    int podSeq = -1;
    {
        std::lock_guard<std::mutex> lockGuard(mutex_);
        if (!sClientIP.empty()) {
            auto podSeqIterator = ipPodSeqMap_.find(sClientIP);
            if (podSeqIterator != ipPodSeqMap_.end()) {
                podSeq = podSeqIterator->second;
            }
        }
    }

    std::ostringstream stream;
    stream << "/apis/k8s.tars.io/v1alpha1/namespaces/" << K8SParams::instance().bindNamespace()
           << "/tconfigs?labelSelector="
           << "tars.io/ServerApp=" << sServerApp
           << ",tars.io/ServerName=" << sSeverName
           << ",tars.io/ConfigName=" << sConfigName
           << ",tars.io/Activated=true"
           << ",!tars.io/Deleting";
    if (podSeq == -1) {
        stream << ",tars.io/PodSeq=m";
    } else {
        stream << ",tars.io/PodSeq+in+(m," << podSeq << ")";
    }

    constexpr int MAX_RETRIES_LOAD_TIMES = 3;
    for (int i = 0; i < MAX_RETRIES_LOAD_TIMES; ++i) {
        usleep(i * 30 * 1000);
        auto k8sClientRequest = K8SClient::instance().postRequest(K8SClientRequestMethod::Get, stream.str(), "");
        bool bTaskFinish = k8sClientRequest->waitFinish(std::chrono::seconds(5));
        if (!bTaskFinish) {
            sErrorInfo.append(k8sClientRequest->stateMessage());
            continue;
        }

        if (k8sClientRequest->state() != Done) {
            sErrorInfo.append(k8sClientRequest->stateMessage());
            continue;
        }

        const auto &responseJson = k8sClientRequest->responseJson();
        auto pItem = rapidjson::GetValueByPointer(responseJson, "/items");
        if (pItem == nullptr) {
            sErrorInfo.append(k8sClientRequest->responseBody(), k8sClientRequest->responseSize());
            continue;
        }

        assert(pItem->IsArray());

        const char *masterConfigContent{};
        size_t masterConfigContentLength{};

        const char *slaveConfigContent{};
        size_t slaveConfigContentLength{};

        bool existDeactivateMasterConfig{false};
        bool existDeactivateSlaveConfig{false};

        auto &&jsonArray = pItem->GetArray();

        for (auto &&item : jsonArray) {
            auto pConfigContent = rapidjson::GetValueByPointer(item, "/configContent");
            assert(pConfigContent != nullptr && pConfigContent->IsString());

            auto pPodSeq = rapidjson::GetValueByPointer(item, "/podSeq");
            auto pConfigDeactivate = rapidjson::GetValueByPointer(item, "/metadata/labels/Deactivate");

            if (pPodSeq == nullptr) {
                if (pConfigDeactivate != nullptr) {
                    existDeactivateMasterConfig = true;
                } else {
                    masterConfigContent = pConfigContent->GetString();
                    masterConfigContentLength = pConfigContent->GetStringLength();
                }
                continue;
            }

            assert(pPodSeq != nullptr);
            if (pConfigDeactivate != nullptr) {
                existDeactivateSlaveConfig = true;
            } else {
                slaveConfigContent = pConfigContent->GetString();
                slaveConfigContentLength = pConfigContent->GetStringLength();
            }
        }

        if (masterConfigContent != nullptr) {
            if (slaveConfigContent != nullptr) {
                sConfigContent.append(masterConfigContent, masterConfigContentLength).append(
                        MultiConfigSeparator).append(slaveConfigContent, slaveConfigContentLength);
                return ConfigInfoInterface::Success;
            }

            assert(slaveConfigContent == nullptr);
            if (!existDeactivateSlaveConfig) {
                stream.str("");
                stream << std::string(masterConfigContent, masterConfigContentLength);
                sConfigContent = stream.str();
                return ConfigInfoInterface::Success;
            }
            continue;
        }
        if (existDeactivateMasterConfig) {
            continue;
        }
        sErrorInfo = "config not existed or not activated";
        return ConfigInfoInterface::ConfigError;
    }

    return ConfigInfoInterface::K8SError;
}

ConfigInfoInterface::GetConfigResult
ConfigInfoInterface::listConfig(const std::string &sServerApp, const std::string &sSeverName,
                                std::vector<std::string> &vector, std::string &sErrorInfo) {
    if (sSeverName.empty()) {
        return listAppConfig(sServerApp, vector, sErrorInfo);
    }
    return listServerConfig(sServerApp, sSeverName, vector, sErrorInfo);
}

ConfigInfoInterface::GetConfigResult
ConfigInfoInterface::listAppConfig(const std::string &sServerApp, std::vector<std::string> &vector,
                                   std::string &sErrorInfo) {

    assert(!sServerApp.empty());

    vector.clear();
    std::ostringstream stream;
    stream << "/apis/k8s.tars.io/v1alpha1/namespaces/" << K8SParams::instance().bindNamespace()
           << "/tconfigs?labelSelector="
           << "tars.io/ServerApp=" << sServerApp
           << ",tars.io/ServerName="
           << ",tars.io/Activated=true"
           << ",!tars.io/Deleting";

    constexpr int MAX_RETRIES_LOAD_TIMES = 3;

    for (int i = 0; i < MAX_RETRIES_LOAD_TIMES; ++i) {
        auto k8sClientRequest = K8SClient::instance().postRequest(K8SClientRequestMethod::Get, stream.str(), "");
        bool bTaskFinish = k8sClientRequest->waitFinish(std::chrono::seconds(5));
        if (!bTaskFinish) {
            sErrorInfo.append(k8sClientRequest->stateMessage());
            continue;
        }

        if (k8sClientRequest->state() != Done) {
            sErrorInfo.append(k8sClientRequest->stateMessage());
            continue;
        }

        const auto &responseJson = k8sClientRequest->responseJson();
        auto pItem = rapidjson::GetValueByPointer(responseJson, "/items");
        if (pItem == nullptr) {
            sErrorInfo.append(k8sClientRequest->responseBody(), k8sClientRequest->responseSize());
            continue;;
        }

        std::set<std::string> configNames;
        assert(pItem->IsArray());
        for (auto &&config:pItem->GetArray()) {
            auto pConfigName = rapidjson::GetValueByPointer(config, "/configName");
            assert(pConfigName != nullptr && pConfigName->IsString());
            configNames.emplace(SFromP(pConfigName));
        }
        vector.assign(configNames.begin(), configNames.end());
        return ConfigInfoInterface::Success;
    }
    return ConfigInfoInterface::K8SError;
}

ConfigInfoInterface::GetConfigResult
ConfigInfoInterface::listServerConfig(const std::string &sServerApp, const std::string &sServerName,
                                      std::vector<std::string> &vector, std::string &sErrorInfo) {

    assert(!sServerApp.empty());
    assert(!sServerName.empty());

    vector.clear();
    std::ostringstream stream;
    stream << "/apis/k8s.tars.io/v1alpha1/namespaces/" << K8SParams::instance().bindNamespace()
           << "/tconfigs?labelSelector=tars.io/ServerApp=" << sServerApp
           << ",tars.io/ServerName=" << sServerName
           << ",tars.io/Activated=true"
           << ",!tars.io/Deleting"
           << ",tars.io/PodSeq=m";

    constexpr int MAX_RETRIES_LOAD_TIMES = 3;

    for (int i = 0; i < MAX_RETRIES_LOAD_TIMES; ++i) {
        auto k8sClientRequest = K8SClient::instance().postRequest(K8SClientRequestMethod::Get, stream.str(), "");
        bool bTaskFinish = k8sClientRequest->waitFinish(std::chrono::seconds(5));
        if (!bTaskFinish) {
            sErrorInfo.append(k8sClientRequest->stateMessage());
            continue;
        }

        if (k8sClientRequest->state() != Done) {
            sErrorInfo.append(k8sClientRequest->stateMessage());
            continue;
        }

        const auto &responseJson = k8sClientRequest->responseJson();
        auto pItem = rapidjson::GetValueByPointer(responseJson, "/items");
        if (pItem == nullptr) {
            sErrorInfo.append(k8sClientRequest->responseBody(), k8sClientRequest->responseSize());
            continue;
        }

        std::set<std::string> configNames;
        assert(pItem->IsArray());
        for (auto &&config:pItem->GetArray()) {
            auto pConfigName = rapidjson::GetValueByPointer(config, "/configName");
            assert(pConfigName != nullptr && pConfigName->IsString());
            configNames.emplace(SFromP(pConfigName));
        }
        vector.assign(configNames.begin(), configNames.end());
        return ConfigInfoInterface::Success;
    }
    return ConfigInfoInterface::K8SError;
}
