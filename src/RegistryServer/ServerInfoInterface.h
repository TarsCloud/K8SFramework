
#pragma once

#include <mutex>
#include <unordered_map>
#include <algorithm>
#include <cassert>
#include <servant/EndpointF.h>
#include "K8SWatcher.h"
#include "util/tc_config.h"
#include "rapidjson/document.h"
#include "Registry.h"

struct PodStatus {
    std::string name;
    std::string podIP;
    std::string hostIP;
    std::string presentState;
};

struct Adapter {
    int port;
    int hostPort;
    std::string name;
    uint thread;
    uint connection;
    uint timeout;
    uint capacity;
    bool isTcp;
    bool isTars;
};

struct UpChain {
    std::unordered_map<string, std::vector<EndpointF>> customUpChain;
    std::vector<EndpointF> defaultUpChain;
};

struct TarsInfo {
    int asyncThread;
    std::string profileContent;
    std::string templateName;
    std::vector<std::shared_ptr<Adapter>> adapters;
};

enum class ServerSubType {
    Tars,
    Normal,
};

struct ServerInfo {
    ServerSubType subType{};
    std::string serverApp{};
    std::string serverName{};
    std::shared_ptr<TarsInfo> tarsInfo{};
    std::vector<std::shared_ptr<PodStatus>> pods{};
};

struct Template {
    std::string content_;
    std::string parent_;
};

class ServerInfoInterface {
private:
    std::mutex mutex_;
    std::shared_ptr<UpChain> upChainInfo_;
    std::unordered_map<std::string, std::shared_ptr<ServerInfo>> serverInfoMap_;  //记录 ${ServerApp}-${ServerName} 与 ServerInfo 的对应关系
    std::unordered_map<std::string, std::shared_ptr<Template>> templateMap_;

public:

    static ServerInfoInterface &instance() {
        static ServerInfoInterface endpointInterface;
        return endpointInterface;
    };

    void onEndpointAdd(const rapidjson::Value &pDocument);

    void onEndpointUpdate(const rapidjson::Value &pDocument);

    void onEndpointDeleted(const rapidjson::Value &pDocument);

    void onTemplateAdd(const rapidjson::Value &pDocument);

    void onTemplateUpdate(const rapidjson::Value &pDocument);

    void onTemplateDeleted(const rapidjson::Value &pDocument);

    void findEndpoint(const string &id, vector<EndpointF> *pActiveEp, vector<EndpointF> *pInactiveEp);

    int getServerDescriptor(const string &serverApp, const string &serverName, ServerDescriptor &descriptor);

    void loadUpChainConf();

private:
    TC_Config getTemplateContent(const std::string &sTemplateName, std::string &result);

    bool joinParentTemplate(const string &sTemplateName, TC_Config &conf, std::string &result);

    int getTarsServerDescriptor(const shared_ptr<ServerInfo> &serverInfo, ServerDescriptor &descriptor);

    void findTarsEndpoint(const std::shared_ptr<ServerInfo> &serverInfo, const string &sPortName, vector<EndpointF> *pActiveEp, vector<EndpointF> *pInactiveEp);

    void findUpChainEndpoint(const std::string &id, vector<EndpointF> *pActiveEp, vector<EndpointF> *pInactiveEp);
};

inline void handleEndpointsEvent(K8SWatchEvent eventType, const rapidjson::Value &pDocument) {

    assert(eventType == K8SWatchEventAdded || eventType == K8SWatchEventDeleted || eventType == K8SWatchEventUpdate);

    if (eventType == K8SWatchEventAdded) {
        return ServerInfoInterface::instance().onEndpointAdd(pDocument);
    }

    if (eventType == K8SWatchEventUpdate) {
        return ServerInfoInterface::instance().onEndpointUpdate(pDocument);
    }

    if (eventType == K8SWatchEventDeleted) {
        return ServerInfoInterface::instance().onEndpointDeleted(pDocument);
    }

    assert(false);
}

inline void handleTemplateEvent(K8SWatchEvent eventType, const rapidjson::Value &pDocument) {

    assert(eventType == K8SWatchEventAdded || eventType == K8SWatchEventDeleted || eventType == K8SWatchEventUpdate);

    if (eventType == K8SWatchEventAdded) {
        return ServerInfoInterface::instance().onTemplateAdd(pDocument);
    }

    if (eventType == K8SWatchEventUpdate) {
        return ServerInfoInterface::instance().onTemplateUpdate(pDocument);
    }

    if (eventType == K8SWatchEventDeleted) {
        return ServerInfoInterface::instance().onTemplateDeleted(pDocument);
    }

    assert(false);
}
