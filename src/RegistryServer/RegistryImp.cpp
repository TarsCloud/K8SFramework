#include "RegistryImp.h"
#include "util/tc_config.h"
#include "servant/RemoteLogger.h"
#include "Storage.h"
#include <K8SParams.h>
#include <K8SClient.h>

void RegistryImp::initialize()
{
}

static int joinTemplates(const std::map<std::string, std::shared_ptr<TTemplate>>& templates, const std::string& name, std::string& result)
{
	TC_Config conf{};
	if (!result.empty())
	{
		try
		{
			conf.parseString(result);
		}
		catch (const TC_Config_Exception& ex)
		{
			TLOGERROR("parser to tc_config error: " << ex.what() << ", content: " << result.substr(0, 2048));
			return -1;
		}
	}
	auto parentTemplateName = name;
	while (true)
	{
		if (parentTemplateName.empty())
		{
			result = conf.tostr();
			return 0;
		}
		auto iterator = templates.find(parentTemplateName);
		if (iterator == templates.end())
		{
			TLOGERROR("template|" << parentTemplateName << " not exist" << endl);
			return -1;
		}
		auto pTemplate = iterator->second;
		TC_Config parentConf{};
		try
		{
			parentConf.parseString(pTemplate->content);
			conf.joinConfig(parentConf, false);
		}
		catch (const TC_Config_Exception& ex)
		{
			TLOGERROR("parser to tc_config error: " << ex.what() << ", content: " << result.substr(0, 2048));
			result = ex.what();
			return -1;
		}
		if (parentTemplateName == pTemplate->parent)
		{
			result = conf.tostr();
			return 0;
		}
		parentTemplateName = pTemplate->parent;
	}
}

Int32
RegistryImp::getServerDescriptor(const std::string& serverApp, const std::string& serverName, ServerDescriptor& serverDescriptor, CurrentPtr current)
{
	int res = 0;
	std::string serverTemplate{};
	Storage::instance().getTEndpoints(
			[serverApp, serverName, &res, &serverDescriptor, &serverTemplate](
					const std::map<std::string, std::shared_ptr<TEndpoint>>& tendpoints)mutable
			{
				auto id = TC_Common::lower(serverApp) + "-" + TC_Common::lower(serverName);
				auto iterator = tendpoints.find(id);
				if (iterator == tendpoints.end())
				{
					res = -1;
					TLOGERROR("server|" << serverApp << "." << serverName << " not exist " << endl);
					return;
				}
				if (iterator->second == nullptr)
				{
					res = -1;
					TLOGERROR("server|" << serverApp << "." << serverName << " not exist " << endl);
					return;
				}
				auto&& endpoint = iterator->second;
				serverTemplate = endpoint->templateName;
				serverDescriptor.asyncThreadNum = endpoint->asyncThread;
				serverDescriptor.profile = endpoint->profileContent;
				const auto& adapters = endpoint->tadapters;
				for (const auto& adapter: adapters)
				{
					AdapterDescriptor ad{};
					ad.adapterName.append(serverApp).append(".").append(serverName).append(".").append(adapter->name).append("Adapter");
					ad.servant.append(serverApp).append(".").append(serverName).append(".").append(adapter->name);
					ad.protocol = adapter->isTars ? "tars" : "not_tars";
					ad.endpoint.append(adapter->isTcp ? "tcp" : "udp").append(" -h ${localip} -p ").append(to_string(adapter->port)).append(
							" -t ").append(to_string(adapter->timeout));
					ad.threadNum = adapter->thread;
					ad.maxConnections = adapter->connection;
					ad.queuecap = adapter->capacity;
					ad.queuetimeout = adapter->timeout;
					serverDescriptor.adapters[ad.adapterName] = ad;
				}
			});

	if (res != 0)
	{
		return res;
	}

	Storage::instance().getTTemplates(
			[serverTemplate, &res, &serverDescriptor](const std::map<std::string, std::shared_ptr<TTemplate>>& ttemplates)mutable
			{
				res = joinTemplates(ttemplates, serverTemplate, serverDescriptor.profile);
			});
	return res;
}

void RegistryImp::updateServerState(const std::string& podName, const std::string& settingState, const std::string& presentState, CurrentPtr current)
{
	std::stringstream strStream;
	strStream.str("");
	strStream << "/api/v1/namespaces/" << K8SParams::Namespace() << "/pods/" << podName << "/status";
	const std::string patchUrl = strStream.str();

	strStream.str("");
	strStream << R"({"status":{"conditions":[{"type":"tars.io/active")" << ","
			  << R"("status":")" << ((settingState == "Active" && presentState == "Active") ? "True" : "False") << R"(",)"
			  << R"("reason":")" << settingState << "/" << presentState << R"("}]}})";
	const std::string patchBody = strStream.str();
	constexpr int MAX_RETRIES_TIMES = 5;
	for (auto i = 0; i < MAX_RETRIES_TIMES; ++i)
	{
		auto patchRequest = K8SClient::instance().postRequest(K8SClientRequestMethod::StrategicMergePatch, patchUrl, patchBody);
		bool finish = patchRequest->waitFinish(std::chrono::milliseconds(1500));
		if (!finish)
		{
			FDLOG("readiness") << "update pod readiness error, " << podName << "|" << settingState << "|" << presentState << ", reason: "
							   << "overtime|1500" << std::endl;
			continue;
		}
		if (patchRequest->state() != Done)
		{
			FDLOG("readiness") << "update pod readiness error, " << podName << "|" << settingState << "|" << presentState << ", reason: "
							   << patchRequest->stateMessage() << std::endl;
			continue;
		}
		constexpr int HTTP_OK = 200;
		if (patchRequest->responseCode() != HTTP_OK)
		{
			FDLOG("readiness") << "update pod readiness error, " << podName << "|" << settingState << "|" << presentState << ", response: \n\t"
							   << patchRequest->responseBody() << std::endl;
			continue;
		}
		FDLOG("readiness") << "update pod readiness success, " << podName << "|" << settingState << "|" << presentState << std::endl;
		return;
	}
	FDLOG("readiness") << "update pod readiness error, this is<<" << MAX_RETRIES_TIMES << "th try, request will discard" << std::endl;
}
