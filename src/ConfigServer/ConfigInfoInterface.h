#pragma once

#include <mutex>
#include <vector>
#include <memory>
#include <unordered_map>
#include <rapidjson/document.h>

class ConfigInfoInterface {
public:

    enum GetConfigResult {
        Success = 0,
        ConfigError = -1,
        K8SError = -2,
    };

private:

    ConfigInfoInterface() = default;

    std::mutex mutex_;
    std::unordered_map<std::string, int> ipPodSeqMap_;

public:
    static ConfigInfoInterface &instance() {
        static ConfigInfoInterface infoInterface;
        return infoInterface;
    }

    void onPodAdd(const rapidjson::Value &pDocument);

    void onPodUpdate(const rapidjson::Value &pDocument);

    void onPodDelete(const rapidjson::Value &pDocument);

    GetConfigResult
    loadConfig(const std::string &sServerApp, const std::string &sSeverName, const std::string &sConfigName,
               const std::string &sClientIP, std::string &sConfigContent, std::string &sErrorInfo);

    GetConfigResult
    listConfig(const std::string &sSeverApp, const std::string &sSeverName, std::vector<std::string> &vector,
               std::string &sErrorInfo);

private:
    GetConfigResult
    loadAppConfig(const std::string &sServerApp, const std::string &sConfigName, std::string &sConfigContent,
                  std::string &sErrorInfo);

    GetConfigResult
    loadServerConfig(const std::string &sServerApp, const std::string &sSeverName, const std::string &sConfigName,
                     const std::string &sClientIP, std::string &sConfigContent, std::string &sErrorInfo);

    GetConfigResult
    listAppConfig(const std::string &sServerApp, std::vector<std::string> &vector, std::string &sErrorInfo);

    GetConfigResult
    listServerConfig(const std::string &sServerApp, const std::string &sServerName, std::vector<std::string> &vector,
                     std::string &sErrorInfo);
};
