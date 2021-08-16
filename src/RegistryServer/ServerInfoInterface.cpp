
#include "ServerInfoInterface.h"
#include "RegistryServer.h"
#include "rapidjson/pointer.h"
#include <thread>

static inline std::string SFromP(const rapidjson::Value *p) {
    assert(p != nullptr);
    return {p->GetString(), p->GetStringLength()};
}

void ServerInfoInterface::findEndpoint(const string &id, vector<EndpointF> *pActiveEp, vector<EndpointF> *pInactiveEp) {
    std::vector<std::string> v = TC_Common::sepstr<string>(id, ".");
    if (v.size() != 3) {
        return;
    }

    const auto sAppServer = TC_Common::lower(v[0]) + "-" + TC_Common::lower(v[1]);
    const auto &sPortName = v[2];

    assert(pActiveEp != nullptr);

    pActiveEp->clear();
    if (pInactiveEp != nullptr) {
        pInactiveEp->clear();
    }

    std::lock_guard<std::mutex> lockGuard(mutex_);

    auto iterator = serverInfoMap_.find(sAppServer);

    if (iterator == serverInfoMap_.end()) {
        return findUpChainEndpoint(id, pActiveEp, pInactiveEp);
    }

    const auto &serverInfo = iterator->second;

    if (serverInfo == nullptr) {
        LOG->debug() << iterator->first << "->serverInfo is nullptr" << endl;
        return;
    }

    switch (serverInfo->subType) {
        case ServerSubType::Tars:
            return findTarsEndpoint(serverInfo, sPortName, pActiveEp, pInactiveEp);
        case ServerSubType::Normal:
            return;
    }
}

static std::shared_ptr<TarsInfo> buildTarsInfoFromDocument(const rapidjson::Value &pDocument) {

    auto pTarsInfo = std::make_shared<TarsInfo>();

    auto pAsyncThread = rapidjson::GetValueByPointer(pDocument, "/spec/tars/asyncThread");
    if (pAsyncThread == nullptr) {
        //fixme  should log
        return nullptr;
    }
    assert(pAsyncThread->IsInt());
    pTarsInfo->asyncThread = pAsyncThread->GetInt();

    auto pProfile = rapidjson::GetValueByPointer(pDocument, "/spec/tars/profile");
    if (pProfile == nullptr) {
        //fixme  should log
        return nullptr;
    }
    assert(pProfile->IsString());
    pTarsInfo->profileContent = SFromP(pProfile);

    auto pTemplate = rapidjson::GetValueByPointer(pDocument, "/spec/tars/template");
    if (pTemplate == nullptr) {
        //fixme  should log
        return nullptr;
    }
    assert(pTemplate->IsString());
    pTarsInfo->templateName = SFromP(pTemplate);

    auto pServants = rapidjson::GetValueByPointer(pDocument, "/spec/tars/servants");
    if (pServants == nullptr) {
        //fixme  should log
        return nullptr;
    }

    assert(pServants->IsArray());
    for (const auto &v :pServants->GetArray()) {
        auto pAdapter = std::make_shared<Adapter>();
        auto pName = rapidjson::GetValueByPointer(v, "/name");
        assert(pName != nullptr && pName->IsString());
        pAdapter->name = SFromP(pName);

        auto pPort = rapidjson::GetValueByPointer(v, "/port");
        assert(pPort != nullptr && pPort->IsInt());
        pAdapter->port = pPort->GetInt();

        auto pThread = rapidjson::GetValueByPointer(v, "/thread");
        assert(pThread != nullptr && pThread->IsInt());
        pAdapter->thread = pThread->GetInt();

        auto pConnection = rapidjson::GetValueByPointer(v, "/connection");
        assert(pConnection != nullptr && pConnection->IsInt());
        pAdapter->connection = pConnection->GetInt();

        auto pTimeout = rapidjson::GetValueByPointer(v, "/timeout");
        assert(pTimeout != nullptr && pTimeout->IsInt());
        pAdapter->timeout = pTimeout->GetInt();

        auto pCapacity = rapidjson::GetValueByPointer(v, "/capacity");
        assert(pCapacity != nullptr && pCapacity->IsInt());
        pAdapter->capacity = pCapacity->GetInt();

        auto pIsTars = rapidjson::GetValueByPointer(v, "/isTars");
        assert(pIsTars != nullptr && pIsTars->IsBool());
        pAdapter->isTars = pIsTars->GetBool();

        auto pIsTCP = rapidjson::GetValueByPointer(v, "/isTcp");
        assert(pIsTCP != nullptr && pIsTCP->IsBool());
        pAdapter->isTcp = pIsTCP->GetBool();

        pTarsInfo->adapters.push_back(pAdapter);
    }

    auto pHostPorts = rapidjson::GetValueByPointer(pDocument, "/spec/hostPorts");

    if (pHostPorts != nullptr) {

        assert(pHostPorts->IsArray());

        for (const auto &hostPort :pHostPorts->GetArray()) {

            auto pNameRef = rapidjson::GetValueByPointer(hostPort, "/nameRef");
            assert(pNameRef != nullptr && pNameRef->IsString());
            auto nameRef = SFromP(pNameRef);

            for (auto &adapter:pTarsInfo->adapters) {

                if (adapter->name == nameRef) {
                    auto pPort = rapidjson::GetValueByPointer(hostPort, "/port");
                    assert(pPort != nullptr && pPort->IsInt());
                    adapter->hostPort = pPort->GetInt();
                    break;
                }
            }
        }
    }

    return pTarsInfo;
}

static std::shared_ptr<ServerInfo> buildServerInfoFromDocument(const rapidjson::Value &pDocument) {

    auto pServerInfo = std::make_shared<ServerInfo>();

    auto pServerApp = rapidjson::GetValueByPointer(pDocument, "/spec/app");
    assert(pServerApp != nullptr && pServerApp->IsString());
    pServerInfo->serverApp = SFromP(pServerApp);

    auto pServerName = rapidjson::GetValueByPointer(pDocument, "/spec/server");
    assert(pServerName != nullptr && pServerName->IsString());
    pServerInfo->serverName = SFromP(pServerName);

    auto pSubType = rapidjson::GetValueByPointer(pDocument, "/spec/subType");
    assert(pSubType != nullptr && pSubType->IsString());

    std::string subTypeStr = SFromP(pSubType);

    constexpr char TarsType[] = "tars";
    constexpr char NormalType[] = "normal";

    if (subTypeStr == TarsType) {
        pServerInfo->subType = ServerSubType::Tars;
    } else if (subTypeStr == NormalType) {
        pServerInfo->subType = ServerSubType::Normal;
    } else {
        assert(false);
        return nullptr;
    }

    switch (pServerInfo->subType) {
        case ServerSubType::Tars:
            pServerInfo->tarsInfo = buildTarsInfoFromDocument(pDocument);
            break;
        case ServerSubType::Normal:
            // todo pServerInfo->normalInfo = buildNormalInfoFromDocument(pDocument);
            break;
    }

    auto pPods = rapidjson::GetValueByPointer(pDocument, "/status/pods");
    if (pPods != nullptr) {
        assert(pPods->IsArray());
        for (const auto &pod :pPods->GetArray()) {
            auto pPod = std::make_shared<PodStatus>();

            auto pName = rapidjson::GetValueByPointer(pod, "/name");
            assert(pName != nullptr && pName->IsString());
            pPod->name = SFromP(pName);

            auto pPodIP = rapidjson::GetValueByPointer(pod, "/podIP");
            assert(pPodIP != nullptr && pPodIP->IsString());
            pPod->podIP = SFromP(pPodIP);

            auto pHostIP = rapidjson::GetValueByPointer(pod, "/hostIP");
            assert(pHostIP != nullptr && pHostIP->IsString());
            pPod->hostIP = SFromP(pHostIP);

            auto pPresentState = rapidjson::GetValueByPointer(pod, "/presentState");
            assert(pPresentState != nullptr && pPresentState->IsString());
            pPod->presentState = SFromP(pPresentState);

            pServerInfo->pods.push_back(pPod);
        }
    }
    return pServerInfo;
}

int ServerInfoInterface::getTarsServerDescriptor(const std::shared_ptr<ServerInfo> &serverInfo, ServerDescriptor &descriptor) {

    const auto &tarsInfo = serverInfo->tarsInfo;
    if (tarsInfo == nullptr) {
        return -1;
    }

    const auto &adapters = tarsInfo->adapters;
    descriptor.asyncThreadNum = tarsInfo->asyncThread;

    const auto &sTemplateName = tarsInfo->templateName;
    assert(!sTemplateName.empty());

    string sResult;

    TC_Config templateConf = getTemplateContent(sTemplateName, sResult);

    const auto &profileContent = tarsInfo->profileContent;

    if (profileContent.empty()) {
        descriptor.profile = templateConf.tostr();
    } else {
        TC_Config profileConf{};
        profileConf.parseString(tarsInfo->profileContent);
        profileConf.joinConfig(templateConf, false);
        descriptor.profile = profileConf.tostr();
    }

    for (const auto &adapter:adapters) {
        AdapterDescriptor adapterDescriptor;
        adapterDescriptor.adapterName.append(serverInfo->serverApp).append(".").append(serverInfo->serverName).append(".").append(adapter->name).append("Adapter");
        adapterDescriptor.servant.append(serverInfo->serverApp).append(".").append(serverInfo->serverName).append(".").append(adapter->name);
        adapterDescriptor.protocol = adapter->isTars ? "tars" : "not_tars";
        adapterDescriptor.endpoint.append(adapter->isTcp ? "tcp" : "udp").append(" -h ${localip} -p ").append(to_string(adapter->port)).append(" -t ").append(
                to_string(adapter->timeout));
        adapterDescriptor.threadNum = adapter->thread;
        adapterDescriptor.maxConnections = adapter->connection;
        adapterDescriptor.queuecap = adapter->capacity;
        adapterDescriptor.queuetimeout = adapter->timeout;
        descriptor.adapters[adapterDescriptor.adapterName] = adapterDescriptor;
    }
    return 0;
}

int ServerInfoInterface::getServerDescriptor(const string &serverApp, const string &serverName, ServerDescriptor &descriptor) {
    std::lock_guard<std::mutex> lockGuard(mutex_);
    const std::string sAppServer = TC_Common::lower(serverApp) + "-" + TC_Common::lower(serverName);
    auto iterator = serverInfoMap_.find(sAppServer);
    if (iterator == serverInfoMap_.end()) {
        LOG->error() << "not found" << serverApp << "-" << serverName << endl;
        return -1;
    }

    if (iterator->second == nullptr) {
        LOG->error() << "null point" << serverApp << "-" << serverName << endl;
        return -1;
    }

    const auto &serverInfo = iterator->second;

    switch (serverInfo->subType) {
        case ServerSubType::Tars:
            return getTarsServerDescriptor(serverInfo, descriptor);
        case ServerSubType::Normal:
            return -1;
    }

    assert(false); //should not read here
    return 0;
}

TC_Config ServerInfoInterface::getTemplateContent(const string &sTemplateName, std::string &result) {
    assert(!sTemplateName.empty());
    TC_Config conf{};
    auto iterator = templateMap_.find(sTemplateName);
    if (iterator != templateMap_.end()) {
        const auto &content = iterator->second->content_;
        conf.parseString(content);
        const auto &parent = iterator->second->parent_;

        if (sTemplateName != parent) {
            joinParentTemplate(parent, conf, result);
        }
    }
    return conf;
}

bool ServerInfoInterface::joinParentTemplate(const string &sTemplateName, TC_Config &conf, std::string &result) {
    assert(!sTemplateName.empty());
    auto currentTemplateName = sTemplateName;

    while (true) {
        auto iterator = templateMap_.find(currentTemplateName);

        if (iterator == templateMap_.end()) {
            //todo set result
            return false;
        }

        const auto &content = iterator->second->content_;

        TC_Config currentConf;
        try {
            currentConf.parseString(content);
            conf.joinConfig(currentConf, false);
        } catch (TC_Config_Exception &ex) {
            result = ex.what();
            return -1;
        }

        auto parentTemplateName = iterator->second->parent_;
        if (currentTemplateName == parentTemplateName) {
            return true;
        }

        currentTemplateName = std::move(parentTemplateName);
    }
}

void ServerInfoInterface::onTemplateAdd(const rapidjson::Value &pDocument) {
    auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(pName != nullptr && pName->IsString());

    auto pContent = rapidjson::GetValueByPointer(pDocument, "/spec/content");
    assert(pContent != nullptr && pContent->IsString());

    auto pParent = rapidjson::GetValueByPointer(pDocument, "/spec/parent");
    assert(pContent != nullptr && pContent->IsString());

    auto pTemplate = std::make_shared<Template>();

    pTemplate->content_ = SFromP(pContent);
    pTemplate->parent_ = SFromP(pParent);

    auto name = SFromP(pName);

    std::lock_guard<std::mutex> lockGuard(mutex_);
    auto iterator = templateMap_.find(name);
    if (iterator == templateMap_.end()) {
        templateMap_[name] = pTemplate;
    } else {
        iterator->second.swap(pTemplate);
    }
}

void ServerInfoInterface::onTemplateUpdate(const rapidjson::Value &pDocument) {
    onTemplateAdd(pDocument);
}

void ServerInfoInterface::onTemplateDeleted(const rapidjson::Value &pDocument) {
    auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(pName != nullptr && pName->IsString());
    auto name = SFromP(pName);
    std::lock_guard<std::mutex> lockGuard(mutex_);
    templateMap_.erase(name);
}

void ServerInfoInterface::onEndpointAdd(const rapidjson::Value &pDocument) {
    auto endpointName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(endpointName != nullptr && endpointName->IsString());
    std::string sEndpointName = SFromP(endpointName);

    auto pServerInfo = buildServerInfoFromDocument(pDocument);
    std::lock_guard<std::mutex> lockGuard(mutex_);
    auto iterator = serverInfoMap_.find(sEndpointName);
    if (iterator == serverInfoMap_.end()) {
        serverInfoMap_[sEndpointName] = pServerInfo;
    } else {
        iterator->second.swap(pServerInfo);
    }
}

void ServerInfoInterface::onEndpointUpdate(const rapidjson::Value &pDocument) {
    onEndpointAdd(pDocument);
}

void ServerInfoInterface::onEndpointDeleted(const rapidjson::Value &pDocument) {
    auto endpointName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(endpointName != nullptr && endpointName->IsString());
    std::string sEndpointName = SFromP(endpointName);
    std::lock_guard<std::mutex> lockGuard(mutex_);
    serverInfoMap_.erase(sEndpointName);
}

void
ServerInfoInterface::findTarsEndpoint(const std::shared_ptr<ServerInfo> &serverInfo, const string &sPortName, vector<EndpointF> *pActiveEp,
                                     vector<EndpointF> *pInactiveEp) {

    const auto &tarsInfo = serverInfo->tarsInfo;

    if (tarsInfo == nullptr) {
        LOG->debug() << serverInfo->serverApp << "." << serverInfo->serverName << "->tarsInfo is nullptr" << endl;
        return;
    }

    const auto &adapters = tarsInfo->adapters;

    const auto &pods = serverInfo->pods;

    for (const auto &port:adapters) {
        if (port->name == sPortName) {
            for (const auto &pod : pods) {
                if (pod->presentState == "Active") {
                    EndpointF endpointF;
                    endpointF.port = port->port;
                    endpointF.istcp = port->isTcp;
                    endpointF.timeout = port->timeout;
                    endpointF.host.append(pod->name).append(".").append(TC_Common::lower(serverInfo->serverApp)).append("-").append(TC_Common::lower(serverInfo->serverName));
                    pActiveEp->push_back(endpointF);
                } else if (pInactiveEp != nullptr) {
                    EndpointF endpointF;
                    endpointF.port = port->port;
                    endpointF.istcp = port->isTcp;
                    endpointF.timeout = port->timeout;
                    endpointF.host.append(pod->name).append(".").append(TC_Common::lower(serverInfo->serverApp)).append("-").append(TC_Common::lower(serverInfo->serverName));
                    pInactiveEp->push_back(endpointF);
                }
            }
        }
    }
}

void ServerInfoInterface::loadUpChainConf() {

    constexpr char UpChainConfFile[] = "/etc/upchain/upchain.conf";

    int fok = ::access(UpChainConfFile, F_OK);
    if (fok != 0) {
        std::lock_guard<std::mutex> lockGuard(mutex_);
        if (upChainInfo_ != nullptr) {
            upChainInfo_ = nullptr;
            LOG->debug() << "clear upchainInfo because file \"" << UpChainConfFile << "\"not exist";
        }
        return;
    }

    int rok = ::access(UpChainConfFile, R_OK);
    if (rok != 0) {
        LOG->error() << "permission denied to read " << UpChainConfFile << endl;
        return;
    }

    auto upChainConfContent = TC_File::load2str(UpChainConfFile);
    if (upChainConfContent.empty()) {
        std::lock_guard<std::mutex> lockGuard(mutex_);
        if (upChainInfo_ != nullptr) {
            upChainInfo_ = nullptr;
            LOG->debug() << "clear upchainInfo because file \"" << UpChainConfFile << "\"is empty";
        }
        return;
    }

    TC_Config tcConfig;
    try {
        tcConfig.parseString(upChainConfContent);
    } catch (TC_Config_Exception &e) {
        LOG->error() << "parser file \"" << UpChainConfFile << "\" content catch exception : " << e.what() << endl;
        return;
    }

    auto upChainInfo = std::make_shared<UpChain>();

    std::vector<std::string> domains = tcConfig.getDomainVector("/upchain");
    for (const auto &domain:domains) {
        auto absDomain = string("/upchain/" + domain);
        auto lines = tcConfig.getDomainLine(absDomain);
        std::vector<EndpointF> ev;
        ev.reserve(lines.size());
        for (auto &&line: lines) {
            TC_Endpoint endpoint(line);
            EndpointF f;
            f.host = endpoint.getHost();
            f.port = endpoint.getPort();
            f.timeout = endpoint.getTimeout();
            f.istcp = endpoint.isTcp();
            ev.emplace_back(f);
        }
        if (domain == "default") {
            upChainInfo->defaultUpChain.swap(ev);
        }
        upChainInfo->customUpChain[domain] = std::move(ev);
    }

    std::lock_guard<std::mutex> lockGuard(mutex_);
    if (upChainInfo_ != nullptr) {
        upChainInfo_.swap(upChainInfo);
        return;
    }
    upChainInfo_ = upChainInfo;
    LOG->debug() << "update upchainInfo success" << endl;
}

void ServerInfoInterface::findUpChainEndpoint(const string &id, vector<EndpointF> *pActiveEp, vector<EndpointF> *pInactiveEp) {
    assert(pActiveEp != nullptr);
    pActiveEp->clear();

    if (upChainInfo_ == nullptr) {
        return;
    }

    auto customIterator = upChainInfo_->customUpChain.find(id);
    if (customIterator != upChainInfo_->customUpChain.end()) {
        *pActiveEp = customIterator->second;
        return;
    }

    if (!upChainInfo_->defaultUpChain.empty()) {
        *pActiveEp = upChainInfo_->defaultUpChain;
    }
}


