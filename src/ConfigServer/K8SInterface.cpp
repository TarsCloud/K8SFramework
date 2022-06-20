
#include <sstream>
#include <K8SParams.h>
#include <rapidjson/pointer.h>
#include <set>
#include <K8SClient.h>
#include "K8SInterface.h"
#include "Storage.h"

constexpr int REQUEST_OVERTIME = 3; // seconds
constexpr int MAX_RETIES = 3;
constexpr char MultiConfigSeparator[] = "\r\n\r\n";

static int getHostSeq(const std::string& host)
{
    int seq = -1;
    if (!host.empty())
    {
        Storage::getSeqMap([host, &seq](const std::unordered_map<std::string, int>& seqMap)mutable
        {
            auto iterator = seqMap.find(host);
            if (iterator != seqMap.end())
            {
                seq = iterator->second;
            }
        });
    }
    return seq;
}

static std::shared_ptr<K8SClientRequest> doK8SRequest(const std::string& head)
{
    auto req = K8SClient::instance().postRequest(K8SClientRequestMethod::Get, head, "");
    bool bTaskFinish = req->waitFinish(std::chrono::seconds(REQUEST_OVERTIME));
    if (!bTaskFinish)
    {
        std::ostringstream os;
        os << "request api-server overtime\n, \turl: " << head;
        throw std::runtime_error(os.str());
    }

    if (req->state() == Error)
    {
        std::ostringstream os;
        os << "request api-server error\n, \trequest: " << head << "\n, \t error: " << req->stateMessage();
        throw std::runtime_error(os.str());
    }

    return req;
}

void K8SInterface::listConfig(const std::string& app, const std::string& server, const std::string& host, std::vector<std::string>& vf)
{
    std::ostringstream stream;
    stream << "/apis/k8s.tars.io/v1beta2/namespaces/" << K8SParams::Namespace()
           << "/tconfigs?resourceVersion=0&labelSelector=tars.io/ServerApp=" << app
           << ",tars.io/ServerName=" << server
           << ",tars.io/Activated=true"
           << ",!tars.io/Deleting"
           << ",tars.io/PodSeq=m";

    auto req = doK8SRequest(stream.str());
    const auto& responseJson = req->responseJson();
    auto pItem = rapidjson::GetValueByPointer(responseJson, "/items");
    if (pItem == nullptr)
    {
        std::ostringstream os;
        os << "request api-server got unexpected response: \n, \trequest: " << stream.str() << "\n, \t response: " << req->responseBody();
        throw std::runtime_error(os.str());
    }

    std::set<std::string> configNames{};
    for (auto&& config: pItem->GetArray())
    {
        auto pConfigName = rapidjson::GetValueByPointer(config, "/configName");
        assert(pConfigName != nullptr && pConfigName->IsString());
        configNames.emplace(pConfigName->GetString(), pConfigName->GetStringLength());
    }
    vf.assign(configNames.begin(), configNames.end());
}

void
K8SInterface::loadConfig(const std::string& app, const std::string& server, const std::string& fileName, const std::string& host, std::string& result)
{
    int podSeq = getHostSeq(host);
    std::ostringstream stream;
    stream << "/apis/k8s.tars.io/v1beta2/namespaces/" << K8SParams::Namespace()
           << "/tconfigs?resourceVersion=0&labelSelector="
           << "tars.io/ServerApp=" << app
           << ",tars.io/ServerName=" << server
           << ",tars.io/ConfigName=" << fileName
           << ",tars.io/Activated=true"
           << ",!tars.io/Deleting";
    if (podSeq == -1)
    {
        stream << ",tars.io/PodSeq=m";
    }
    else
    {
        stream << ",tars.io/PodSeq+in+(m," << podSeq << ")";
    }

    for (auto i = 0; i < MAX_RETIES; ++i)
    {
        auto req = doK8SRequest(stream.str());
        const auto& responseJson = req->responseJson();
        auto pItem = rapidjson::GetValueByPointer(responseJson, "/items");
        if (pItem == nullptr)
        {
            std::ostringstream os;
            os << "request api-server got unexpected response: \n, \trequest: " << stream.str() << "\n, \t response: " << req->responseBody();
            throw std::runtime_error(os.str());
        }

        const char* masterConfigContent{};
        const char* slaveConfigContent{};

        bool existDeactivateMasterConfig{ false };
        bool existDeactivateSlaveConfig{ false };

        auto&& jsonArray = pItem->GetArray();
        for (auto&& item: jsonArray)
        {
            auto pConfigContent = rapidjson::GetValueByPointer(item, "/configContent");
            assert(pConfigContent != nullptr && pConfigContent->IsString());

            auto pPodSeq = rapidjson::GetValueByPointer(item, "/podSeq");

            std::string sPodSeq = (pPodSeq == nullptr ? "m" : std::string(pPodSeq->GetString(), pPodSeq->GetStringLength()));

            auto pConfigDeactivate = rapidjson::GetValueByPointer(item, "/metadata/labels/Deactivate");

            if (sPodSeq == "m")
            {
                if (pConfigDeactivate != nullptr)
                {
                    existDeactivateMasterConfig = true;
                }
                else
                {
                    masterConfigContent = pConfigContent->GetString();
                }
                continue;
            }

            if (pConfigDeactivate != nullptr)
            {
                existDeactivateSlaveConfig = true;
            }
            else
            {
                slaveConfigContent = pConfigContent->GetString();
            }
        }
        std::ostringstream os;
        if (masterConfigContent != nullptr)
        {
            if (slaveConfigContent != nullptr)
            {
                os << masterConfigContent << MultiConfigSeparator << slaveConfigContent;
                result = os.str();
                return;
            }
            assert(slaveConfigContent == nullptr);
            if (!existDeactivateSlaveConfig)
            {
                result = masterConfigContent;
                return;
            }
        }
        if (existDeactivateMasterConfig)
        {
            continue;
        }
        throw std::runtime_error("config not existed or not activated");
    }
    throw std::runtime_error("config not existed or not activated");
}

