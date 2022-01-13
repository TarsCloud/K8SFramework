#pragma once

#include "K8SWatcher.h"
#include "servant/EndpointF.h"
#include "util/tc_thread_rwlock.h"
#include <string>
#include <map>
#include <vector>
#include <set>

struct TTemplate
{
    std::string name;
    std::string parent;
    std::string content;
};

struct TAdapter
{
    bool isTcp{};
    bool isTars{};
    int port{};
    int32_t thread{};
    int32_t connection{};
    int32_t timeout{};
    int32_t capacity{};
    std::string name;
};

struct UPChain
{
    std::vector<tars::EndpointF> defaults;
    std::map<std::string, std::vector<tars::EndpointF>> customs;
};

struct TEndpoint
{
    int asyncThread{};
    std::string profileContent;
    std::string templateName;
    std::vector<std::shared_ptr<TAdapter>> tAdapters;
    std::set<std::string> activatedPods;
    std::set<std::string> inActivatedPods;
};

class Storage
{
public:
    static Storage& instance()
    {
        static Storage store;
        return store;
    }

    inline void preListTemplate()
    {
        cacheTTs_.clear();
    }

    inline void postListTemplate()
    {
        {
            ttMutex_.writeLock();
            std::swap(tts_, cacheTTs_);
            ttMutex_.unWriteLock();
        }
        cacheTTs_.clear();
    }

    inline void updateTemplate(const std::string& name, const std::shared_ptr<TTemplate>& t, K8SWatchEventDrive driver)
    {
        if (driver == K8SWatchEventDrive::List)
        {
            cacheTTs_[t->name] = t;
            return;
        }
        ttMutex_.writeLock();
        tts_[t->name] = t;
        ttMutex_.unWriteLock();
    }

    inline void deleteTemplate(const std::string& name)
    {
        ttMutex_.writeLock();
        tts_.erase(name);
        ttMutex_.unWriteLock();
    }

    inline void preListEndpoint()
    {
        cacheTEs_.clear();
    }

    inline void postListEndpoint()
    {
        {
            teMutex_.writeLock();
            std::swap(tes_, cacheTEs_);
            teMutex_.unWriteLock();
        }
        cacheTEs_.clear();
    }

    inline void
    updateEndpoint(const std::string& name, const std::shared_ptr<TEndpoint>& t, K8SWatchEventDrive driver)
    {
        if (driver == K8SWatchEventDrive::List)
        {
            cacheTEs_[name] = t;
            return;
        }
        teMutex_.writeLock();
        tes_[name] = t;
        teMutex_.unWriteLock();
    }

    inline void deleteTEndpoint(const std::string& name)
    {
        teMutex_.writeLock();
        tes_.erase(name);
        teMutex_.unWriteLock();
    }

    inline void updateUpChain(const std::shared_ptr<UPChain>& upChain)
    {
        upChainMutex_.writeLock();
        upChain_ = upChain;
        upChainMutex_.unWriteLock();
    }

    inline void getTEndpoints(const std::function<void(const std::map<std::string, std::shared_ptr<TEndpoint>>&)>& f)
    {
        teMutex_.readLock();
        f(tes_);
        teMutex_.unReadLock();
    }

    inline void getTTemplates(const std::function<void(const std::map<std::string, std::shared_ptr<TTemplate>>&)>& f)
    {
        ttMutex_.readLock();
        f(tts_);
        ttMutex_.unReadLock();
    }

    inline void getUnChain(const std::function<void(std::shared_ptr<UPChain>&)>& f)
    {
        upChainMutex_.readLock();
        f(upChain_);
        upChainMutex_.unReadLock();
    }


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
