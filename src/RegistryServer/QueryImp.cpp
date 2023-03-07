#include "QueryImp.h"
#include "Storage.h"
#include <string>
#include "servant/RemoteLogger.h"

static std::string
buildLogContent(const CurrentPtr& current, const vector<EndpointF>& activeEp, const vector<EndpointF>* inactiveEp)
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
    constexpr int FixedTimeout = 6000;

    constexpr char FixedRegistryServerHost[] = "tars-tarsregistry";

    constexpr char FixedQueryAdapterName[] = "tars.tarsregistry.QueryObj";
    constexpr int FixedQueryAdapterPort = 17890;

    constexpr char FixedRegistryAdapterName[] = "tars.tarsregistry.RegistryObj";
    constexpr int FixedRegistryAdapterPort = 17891;

    std::vector<std::string> v = TC_Common::sepstr<string>(id, ".");
    if (v.size() != 3 || v[0].empty() || v[1].empty() || v[2].empty())
    {
        return;
    }

    if (id == FixedQueryAdapterName)
    {
        EndpointF endpointF{};
        endpointF.host = FixedRegistryServerHost;
        endpointF.port = FixedQueryAdapterPort;
        endpointF.istcp = true;
        endpointF.timeout = FixedTimeout;
        activeEp.push_back(endpointF);
        return;
    }

    if (id == FixedRegistryAdapterName)
    {
        EndpointF endpointF{};
        endpointF.host = FixedRegistryServerHost;
        endpointF.port = FixedRegistryAdapterPort;
        endpointF.istcp = true;
        endpointF.timeout = FixedTimeout;
        activeEp.push_back(endpointF);
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
                    const std::map<std::string, std::shared_ptr<TEndpoint>>& endpoints) mutable
            {
                auto endpointName = tars::TC_Common::lower(v[0]) + "-" + tars::TC_Common::lower(v[1]);
                auto iterator = endpoints.find(endpointName);
                if (iterator == endpoints.end())
                {
                    return;
                }
                serviceExistInCluster = true;
                const auto& endpoint = iterator->second;
                if (endpoint == nullptr)
                {
                    return;
                }
                for (auto&& servant: endpoint->servants)
                {
                    if (servant.name == v[2])
                    {
                        EndpointF endpointF{};
                        endpointF.port = servant.port;
                        endpointF.istcp = servant.isTcp;
                        endpointF.timeout = servant.timeout;
                        for (const auto& pod: endpoint->activatedPods)
                        {
                            endpointF.host.append(pod).append(".").append(endpointName);
                            activeEp.push_back(endpointF);
                        }
                        if (inactiveEp != nullptr)
                        {
                            for (auto&& pod: endpoint->inActivatedPods)
                            {
                                endpointF.host.append(pod).append(".").append(endpointName);
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
            activeEp = iterator->second;
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

int QueryImp::findObjectById4Any(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp,
        CurrentPtr current)
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


int QueryImp::findObjectById4All(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp,
        CurrentPtr current)
{
    findObjectById_(id, activeEp, &inactiveEp);
    auto str = buildLogContent(current, activeEp, &inactiveEp);
    DAY_LOG("query_idc", __FUNCTION__, id, "", str);
    return 0;
}

int
QueryImp::findObjectByIdInSameGroup(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp,
        CurrentPtr current)
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
