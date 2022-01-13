#include "QueryImp.h"
#include "Storage.h"
#include <string>
#include "servant/RemoteLogger.h"

static std::string buildLogContent(const CurrentPtr& current, const vector<EndpointF>& activeEp, const vector<EndpointF>* inactiveEp)
{
    std::ostringstream os;
    if (current)
    {
        os << "|" << current->getIp() << ":" << current->getPort();
    }
    os << "|";
    for (size_t i = 0; i < activeEp.size(); i++)
    {
        if (0 != i)
        {
            os << ";";
        }
        os << activeEp[i].host << ":" << TC_Common::tostr(activeEp[i].port);
    }

    os << "|";
    if (inactiveEp != nullptr)
    {
        for (size_t i = 0; i < inactiveEp->size(); i++)
        {
            if (0 != i)
            {
                os << ";";
            }
            os << (*inactiveEp)[i].host + ":" + TC_Common::tostr((*inactiveEp)[i].port);
        }
    }
    os << "|";
    return os.str();
}

#define DAY_LOG(A, B, C, D, E)                           \
do{                                                      \
 FDLOG(A)<<"|"<<(B)<<"|"<<(C)<<"|"<<(D)<<"|"<<(E)<<endl; \
}                                                        \
while(false)                                             \


static void findObjectById_(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>* inactiveEp)
{
    std::vector<std::string> v = TC_Common::sepstr<string>(id, ".");
    if (v.size() != 3)
    {
        return;
    }

    bool serviceExistInCluster = false;
/*
	we must distinguish between two situations:
		1. service exists in k8s cluster but there are no active nodes,
		2. service does not exist in k8s cluster
 */

    Storage::instance().getTEndpoints(
            [&v, &activeEp, &inactiveEp, &serviceExistInCluster](
                    const std::map<std::string, std::shared_ptr<TEndpoint>>& tendpoints) mutable
            {
                auto tendpointName = TC_Common::lower(v[0] + "-" + v[1]);
                auto iterator = tendpoints.find(tendpointName);
                if (iterator == tendpoints.end())
                {
                    return;
                }
                serviceExistInCluster = true;
                const auto& tendpoint = iterator->second;
                if (tendpoint == nullptr)
                {
                    return;
                }
                for (auto& tadapter: tendpoint->tAdapters)
                {
                    if (tadapter->name == v[2])
                    {
                        EndpointF endpointF{};
                        endpointF.port = tadapter->port;
                        endpointF.istcp = tadapter->isTcp;
                        endpointF.timeout = tadapter->timeout;
                        for (const auto& pod: tendpoint->activatedPods)
                        {
                            endpointF.host = std::string{}.append(pod).append(".").append(tendpointName);
                            activeEp.push_back(endpointF);
                        }
                        if (inactiveEp != nullptr)
                        {
                            for (auto&& pod: tendpoint->inActivatedPods)
                            {
                                endpointF.host = std::string{}.append(pod).append(".").append(tendpointName);
                                inactiveEp->push_back(endpointF);
                            }
                        }
                        return;
                    }
                }
            });

    if (serviceExistInCluster)
    {
        return;
    }

    Storage::instance().getUnChain([&id, &activeEp](const std::shared_ptr<UPChain>& upChain)mutable
    {
        if (upChain == nullptr)
        {
            return;
        }
        auto iterator = upChain->customs.find(id);
        if (iterator != upChain->customs.end())
        {
            activeEp.emplace_back(

            );
            return;
        }
        if (!upChain->defaults.empty())
        {
            activeEp = upChain->defaults;
            return;
        }
    });
}

vector<EndpointF> QueryImp::findObjectById(const std::string& id, CurrentPtr current)
{
    vector<EndpointF> activeEp;
    findObjectById_(id, activeEp, nullptr);
    auto str = buildLogContent(current, activeEp, nullptr);
    DAY_LOG("query", __FUNCTION__, id, "", str);
    return activeEp;
}

int QueryImp::findObjectById4Any(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp, CurrentPtr current)
{
    findObjectById_(id, activeEp, &inactiveEp);
    auto str = buildLogContent(current, activeEp, &inactiveEp);
    DAY_LOG("query", __FUNCTION__, id, "", str);
    return 0;
}

Int32
QueryImp::findObjectByIdInSameStation(const std::string& id, const std::string& sStation, vector<EndpointF>& activeEp,
        vector<EndpointF>& inactiveEp,
        CurrentPtr current)
{
    findObjectById_(id, activeEp, &inactiveEp);
    auto str = buildLogContent(current, activeEp, &inactiveEp);
    DAY_LOG("query", __FUNCTION__, id, "", str);
    return 0;
}


int QueryImp::findObjectById4All(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp, CurrentPtr current)
{
    findObjectById_(id, activeEp, &inactiveEp);
    auto str = buildLogContent(current, activeEp, &inactiveEp);
    DAY_LOG("query_idc", __FUNCTION__, id, "", str);
    return 0;
}

int
QueryImp::findObjectByIdInSameGroup(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp, CurrentPtr current)
{
    findObjectById_(id, activeEp, &inactiveEp);
    auto str = buildLogContent(current, activeEp, &inactiveEp);
    DAY_LOG("query_idc", __FUNCTION__, id, "", str);
    return 0;
}


Int32
QueryImp::findObjectByIdInSameSet(const std::string& id, const std::string& setId, vector<EndpointF>& activeEp,
        vector<EndpointF>& inactiveEp,
        CurrentPtr current)
{
    findObjectById_(id, activeEp, &inactiveEp);
    auto str = buildLogContent(current, activeEp, &inactiveEp);
    DAY_LOG("query_set", __FUNCTION__, id, setId, str);
    return 0;
}