#pragma once

#include "K8SWatcher.h"
#include "servant/EndpointF.h"
#include "util/tc_thread_rwlock.h"
#include <string>
#include <map>
#include <vector>
#include <set>
#include "TEndpoint.h"
#include "TServer.h"
#include "TServant.h"
#include "TTemplate.h"
#include "TFrameworkConfig.h"

class Storage
{
public:
    static Storage& instance()
    {
        static Storage store;
        return store;
    }

    void preListTTemplate();

    void postListTTemplate();

    void updateTTemplate(const std::string& name, const std::shared_ptr<TTemplate>& t, K8SWatchEventDrive driver);

    void deleteTTemplate(const std::string& name);

    void getTTemplates(const std::function<void(const std::map<std::string, std::shared_ptr<TTemplate>>&)>& f);


    void preListTEndpoint();

    void postListTEndpoint();

    void updateTEndpoint(const std::string& name, const std::shared_ptr<TEndpoint>& t, K8SWatchEventDrive driver);

    void deleteTEndpoint(const std::string& name);

    void getTEndpoints(const std::function<void(const std::map<std::string, std::shared_ptr<TEndpoint>>&)>& f);


    void updateUpChain(const std::shared_ptr<UPChain>& upChain);

    void getUnChain(const std::function<void(std::shared_ptr<UPChain>&)>& f);

    ~Storage() = default;

    Storage(const Storage&) = delete;

private:
    Storage() = default;

private:
    tars::TC_ThreadRWLocker teMutex_;
    std::map<std::string, std::shared_ptr<TEndpoint>> tes_;
    std::map<std::string, std::shared_ptr<TEndpoint>> cacheTEs_;

    tars::TC_ThreadRWLocker ttMutex_;
    std::map<std::string, std::shared_ptr<TTemplate>> tts_;
    std::map<std::string, std::shared_ptr<TTemplate>> cacheTTs_;

    tars::TC_ThreadRWLocker upChainMutex_;
    std::shared_ptr<UPChain> upChain_;
};
