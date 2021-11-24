#include "RegistryServer.h"
#include "RegistryImp.h"
#include "QueryImp.h"
#include "K8SClient.h"
#include "K8SWatcher.h"
#include "K8SParams.h"
#include "K8SWatchCallback.h"

static void onError(const std::error_code& ec, const std::string& msg)
{
	TLOGERROR(ec.message() << ": " << msg << std::endl);
	std::cout << ec.message() << ": " << msg << std::endl;
	exit(-1);
};

static void createK8SContext()
{
	K8SClient::instance().start();
	K8SWatcher::instance().start();

	K8SWatcherSetting tendpointWatchSetting("k8s.tars.io", "v1beta1", "tendpoints", K8SParams::Namespace());
	tendpointWatchSetting.onAdded = K8SWatchCallback::onEndpointAdded;
	tendpointWatchSetting.onModified = K8SWatchCallback::onEndpointModified;
	tendpointWatchSetting.onDeleted = K8SWatchCallback::onEndpointDeleted;
	tendpointWatchSetting.preList = K8SWatchCallback::prePodList;
	tendpointWatchSetting.postList = K8SWatchCallback::postEndpointList;
	tendpointWatchSetting.onError = onError;

	K8SWatcherSetting ttemplateWatchSetting("k8s.tars.io", "v1beta1", "ttemplates", K8SParams::Namespace());
	ttemplateWatchSetting.preList = K8SWatchCallback::preTemplateList;
	ttemplateWatchSetting.postList = K8SWatchCallback::postTemplateList;
	ttemplateWatchSetting.onAdded = K8SWatchCallback::onTemplateAdded;
	ttemplateWatchSetting.onModified = K8SWatchCallback::onTemplateModified;
	ttemplateWatchSetting.onDeleted = K8SWatchCallback::onTemplateDeleted;
	ttemplateWatchSetting.onError = onError;

	K8SWatcherSetting configWatchSetting("", "v1", "configmaps", K8SParams::Namespace());
	configWatchSetting.setFiledFilter("metadata.name=tars-tarsregistry");
	configWatchSetting.onAdded = K8SWatchCallback::onConfigAdded;
	configWatchSetting.onModified = K8SWatchCallback::onConfigModified;
	configWatchSetting.onDeleted = K8SWatchCallback::onConfigDeleted;
	configWatchSetting.onError = onError;

	K8SWatcher::instance().addWatch(tendpointWatchSetting);
	K8SWatcher::instance().addWatch(ttemplateWatchSetting);
	K8SWatcher::instance().addWatch(configWatchSetting);
}

static void postReadinessGate()
{

	constexpr char PodNameEnv[] = "PodName";

	std::string podName = ::getenv(PodNameEnv);
	if (podName.empty())
	{
		TLOGERROR("Get Empty PodName Value" << std::endl);
		std::cout << "Get Empty PodName Value" << std::endl;
		exit(-1);
	}

	std::stringstream strStream;
	strStream.str("");
	strStream << "/api/v1/namespaces/" << K8SParams::Namespace() << "/pods/" << podName << "/status";
	const std::string setActiveUrl = strStream.str();
	strStream.str("");
	strStream << R"({"status":{"conditions":[{"type":"tars.io/active","status":"True","reason":"Active/Active"}]}})";
	const std::string setActiveBody = strStream.str();

	for (auto i = 0; i < 3; ++i)
	{
		std::this_thread::sleep_for(std::chrono::milliseconds(5));
		auto postReadinessRequest = K8SClient::instance().postRequest(K8SClientRequestMethod::StrategicMergePatch, setActiveUrl, setActiveBody);
		bool finish = postReadinessRequest->waitFinish(std::chrono::seconds(2));
		if (!finish)
		{
			TLOGERROR("Update Registry Server State To \"Active/Active\" Overtime" << std::endl);
			continue;
		}

		if (postReadinessRequest->state() != Done)
		{
			TLOGERROR("Update Registry Server State To \"Active/Active\" Error: " << postReadinessRequest->stateMessage() << std::endl);
			continue;
		}
		TLOGINFO("Update Registry Server State To \"Active/Active\" Success" << std::endl);
		return;
	}
	TLOGERROR("Update Registry Server State To \"Active/Active\" Failed" << std::endl);
	std::cout << "Update Registry Server State To \"Active/Active\" Failed" << std::endl;
	exit(-1);
}

void KEventServer::initialize()
{

	TLOGDEBUG("RegistryServer::initialize..." << std::endl);
	std::cout << "RegistryServer::initialize..." << std::endl;

	createK8SContext();
	if (!K8SWatcher::instance().waitSync(std::chrono::seconds(30)))
	{
		TLOGERROR("RegistryServer Wait K8SWatcher Error Or Overtime|30s" << std::endl);
		std::cout << "RegistryServer Wait K8SWatcher Error Or Overtime|30s" << std::endl;
		exit(-1);
	}

	try
	{
		constexpr char FIXED_QUERY_SERVANT[] = "tars.tarsregistry.QueryObj";
		constexpr char FIXED_REGISTRY_SERVANT[] = "tars.tarsregistry.RegistryObj";
		addServant<QueryImp>(FIXED_QUERY_SERVANT);
		addServant<RegistryImp>(FIXED_REGISTRY_SERVANT);
	}
	catch (TC_Exception& ex)
	{
		TLOGERROR("RegistryServer Add Servant Got Exception: " << ex.what() << std::endl);
		std::cout << "RegistryServer Add Servant Got Exception: " << ex.what() << std::endl;
		exit(-1);
	}

	postReadinessGate();

	TLOGINFO("RegistryServer::initialize OK!" << std::endl);
	std::cout << "RegistryServer::initialize OK!" << std::endl;
}

void KEventServer::destroyApp()
{
	K8SClient::instance().stop();
	K8SWatcher::instance().stop();
	std::this_thread::sleep_for(std::chrono::milliseconds(200));
	TLOGINFO("RegistryServer::destroyApp ok" << std::endl);
	std::cout << "RegistryServer::destroyApp ok" << std::endl;
}
