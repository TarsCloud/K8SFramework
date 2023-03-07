
#include "K8SInterface.h"
#include "K8SParams.h"
#include "K8SClient.h"
#include "Storage.h"
#include "TCodec.h"
#include <util/tc_common.h>
#include <sstream>
#include <set>

constexpr int REQUEST_OVERTIME = 3; // seconds
constexpr int MAX_RETIES = 3;
constexpr char MultiConfigSeparator[] = "\r\n\r\n";

static int getHostSeq(const std::string& host, const std::string& app, const std::string& server)
{
    if (host.empty())
    {
        return -1;
    }

    int seq = -1;
    Storage::getSeqMap([host, &seq](const std::unordered_map<std::string, int>& seqMap)mutable
    {
        auto iterator = seqMap.find(host);
        if (iterator != seqMap.end())
        {
            seq = iterator->second;
        }
    });

    if (seq != -1)
    {
        return seq;
    }

    /*
      if host match pattern “app-server-%d” or “app-server-%d.app-server”,
      we should extract "%d" as pod seq
     */

    auto prefix = tars::TC_Common::lower(app) + "-" + tars::TC_Common::lower(server);
    auto pos = prefix.size();
    if (host.size() - pos < 2)
    {
        return -1;
    }

    if (host.compare(0, pos, prefix) != 0)
    {
        return -1;
    }

    if (host[pos] != '-')
    {
        return -1;
    }

    auto d = 0;
    for (pos += 1; pos != host.size(); ++pos)
    {
        auto c = host[pos];
        if (c >= '0' && c <= '9')
        {
            d = d * 10 + c - '0';
            continue;
        }
        if (c == '.')
        {
            break;
        }
    }

    if (pos == host.size())
    {
        return d;
    }

    if (host.compare(pos + 1, prefix.size(), prefix) != 0)
    {
        return -1;
    }

    return d;
}

static std::shared_ptr <K8SClientRequest> doK8SRequest(const std::string& head)
{
    auto task = K8SClient::instance().postRequest(K8SClientRequestMethod::Get, head, "");
    bool finish = task->waitFinish(std::chrono::seconds(REQUEST_OVERTIME));
    if (!finish)
    {
        std::ostringstream os;
        os << "request api-server overtime\n, \turl: " << head;
        throw std::runtime_error(os.str());
    }

    if (task->state() == Error)
    {
        std::ostringstream os;
        os << "request api-server error\n, \trequest: " << head << "\n, \t error: " << task->stateMessage();
        throw std::runtime_error(os.str());
    }

    return task;
}

void K8SInterface::listConfig(const std::string& app, const std::string& server, const std::string& host,
        std::vector <std::string>& vf)
{
    std::ostringstream stream;
    stream << "/apis/k8s.tars.io/v1beta3/namespaces/" << K8SParams::Namespace()
           << "/tconfigs?labelSelector=tars.io/ServerApp=" << app
           << ",tars.io/ServerName=" << server
           << ",tars.io/Activated=true"
           << ",!tars.io/Deleting"
           << ",tars.io/PodSeq=m";

    auto&& req = doK8SRequest(stream.str());
    auto&& responseJson = req->responseJson();
    auto&& items = responseJson.at("items");

    std::set <std::string> configNames{};
    for (auto&& config: items.get_array())
    {
        auto pConfigName = config.at("configName");
        assert(pConfigName != nullptr && pConfigName.is_string());
        configNames.emplace(boost::json::value_to<std::string>(pConfigName));
    }
    vf.assign(configNames.begin(), configNames.end());
}

void
K8SInterface::loadConfig(const std::string& app, const std::string& server, const std::string& fileName,
        const std::string& host, std::string& result)
{
    int podSeq = getHostSeq(host, app, server);
    std::ostringstream stream;
    stream << "/apis/k8s.tars.io/v1beta3/namespaces/" << K8SParams::Namespace()
           << "/tconfigs?labelSelector="
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
        auto&& req = doK8SRequest(stream.str());
        auto&& responseJson = req->responseJson();
        auto&& items = responseJson.at("items");

        std::string masterConfigContent{};
        std::string slaveConfigContent{};

        bool existDeactivateMasterConfig{ false };
        bool existDeactivateSlaveConfig{ false };

        for (auto&& item: items.get_array())
        {
            VAR_FROM_JSON(std::string, content, item.at("configContent"));
            VAR_FROM_JSON(std::string, seq, item.at("podSeq"));
            boost::system::error_code ec{};
            auto pConfigDeactivate = item.find_pointer("/metadata/labels/Deactivate", ec);
            if (seq == "m")
            {
                if (pConfigDeactivate == nullptr)
                {
                    masterConfigContent.swap(content);
                }
                else
                {
                    existDeactivateMasterConfig = true;
                }
            }
            else
            {
                if (pConfigDeactivate == nullptr)
                {
                    slaveConfigContent.swap(content);
                }
                else
                {
                    existDeactivateSlaveConfig = true;
                }
            }
        }
        if (!masterConfigContent.empty())
        {
            result.swap(masterConfigContent);
            if (!slaveConfigContent.empty())
            {
                result.append(MultiConfigSeparator).append(slaveConfigContent);
                return;
            }
            if (!existDeactivateSlaveConfig)
            {
                return;
            }
        }
        if (existDeactivateMasterConfig || existDeactivateSlaveConfig)
        {
            continue;
        }
        throw std::runtime_error("config not existed or not activated");
    }
    throw std::runtime_error("config not existed or not activated");
}
