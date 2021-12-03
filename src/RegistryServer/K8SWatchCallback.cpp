
#include <map>
#include "K8SWatchCallback.h"
#include "rapidjson/pointer.h"
#include <set>
#include <iostream>
#include <rapidjson/stringbuffer.h>
#include <rapidjson/writer.h>
#include <rapidjson/document.h>
#include <util/tc_clientsocket.h>
#include "servant/EndpointF.h"
#include "util/tc_config.h"
#include "Storage.h"
#include <servant/RemoteLogger.h>

static inline std::string JP2S(const rapidjson::Value* p)
{
	assert(p != nullptr && p->IsString());
	return { p->GetString(), p->GetStringLength() };
}

void GetJsonText(const rapidjson::Value& v)
{
	rapidjson::StringBuffer buffer;

	buffer.Clear();

	rapidjson::Writer<rapidjson::StringBuffer> writer(buffer);
	v.Accept(writer);

	std::cout << buffer.GetString() << std::endl;
}

static std::shared_ptr<TEndpoint> buildTendpointFromDocument(const rapidjson::Value& pDocument)
{
	auto pTEndpoint = std::make_shared<TEndpoint>();
	auto pAsyncThread = rapidjson::GetValueByPointer(pDocument, "/asyncThread");
	if (pAsyncThread == nullptr)
	{
		//fixme  should log
		return nullptr;
	}
	assert(pAsyncThread->IsInt());
	pTEndpoint->asyncThread = pAsyncThread->GetInt();

	auto pProfile = rapidjson::GetValueByPointer(pDocument, "/profile");
	if (pProfile == nullptr)
	{
		//fixme  should log
		return nullptr;
	}
	assert(pProfile->IsString());
	pTEndpoint->profileContent = JP2S(pProfile);

	auto pTemplate = rapidjson::GetValueByPointer(pDocument, "/template");
	if (pTemplate == nullptr)
	{
		//fixme  should log
		return nullptr;
	}
	assert(pTemplate->IsString());
	pTEndpoint->templateName = JP2S(pTemplate);

	auto pServants = rapidjson::GetValueByPointer(pDocument, "/servants");
	if (pServants == nullptr)
	{
		//fixme  should log
		return nullptr;
	}
	assert(pServants->IsArray());
	for (const auto& v: pServants->GetArray())
	{
		auto pAdapter = std::make_shared<TAdapter>();
		auto pName = rapidjson::GetValueByPointer(v, "/name");
		assert(pName != nullptr && pName->IsString());
		pAdapter->name = JP2S(pName);

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

		pTEndpoint->tadapters.push_back(pAdapter);
	}
	return pTEndpoint;
}

static std::shared_ptr <UpChain> buildUpChainFromDocument(const rapidjson::Value& pDocument)
{
	auto pData = rapidjson::GetValueByPointer(pDocument, "/data/upchain.conf");
	if (pData == nullptr)
	{
		return nullptr;
	}
	assert(pData->IsString());
	auto content = JP2S(pData);

	tars::TC_Config tcConfig;
	try
	{
		tcConfig.parseString(content);
	}
	catch (tars::TC_Config_Exception& e)
	{
		TLOGERROR("Parser UPChain Config Content Catch Exception: " << e.what() << std::endl);
		return nullptr;
	}
	auto pUpChain = std::make_shared<UpChain>();
	std::vector <std::string> domains = tcConfig.getDomainVector("/upchain");
	for (const auto& domain: domains)
	{
		auto absDomain = string("/upchain/" + domain);
		auto lines = tcConfig.getDomainLine(absDomain);
		std::vector <tars::EndpointF> ev;
		ev.reserve(lines.size());
		for (auto&& line: lines)
		{
			try
			{
				tars::TC_Endpoint endpoint(line);
				tars::EndpointF f;
				f.host = endpoint.getHost();
				f.port = endpoint.getPort();
				f.timeout = endpoint.getTimeout();
				f.istcp = endpoint.isTcp();
				ev.emplace_back(f);
			}
			catch (const std::exception& e)
			{
				TLOGERROR("Parser UPChain Config Content Catch Exception: " << e.what() << std::endl);
			}
		}
		if (domain == "default")
		{
			pUpChain->defaults.swap(ev);
		}
		else
		{
			pUpChain->customs[domain].swap(ev);
		}
	}
	return pUpChain;
}

void K8SWatchCallback::preTemplateList()
{
	Storage::instance().preListTemplate();
}

void K8SWatchCallback::postTemplateList()
{
	Storage::instance().postListTemplate();
}

void K8SWatchCallback::onTemplateAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver)
{
	auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
	assert(pName != nullptr && pName->IsString());

	auto pContent = rapidjson::GetValueByPointer(pDocument, "/spec/content");
	assert(pContent != nullptr && pContent->IsString());

	auto pParent = rapidjson::GetValueByPointer(pDocument, "/spec/parent");
	assert(pContent != nullptr && pContent->IsString());

	auto pTemplate = std::make_shared<TTemplate>();
	pTemplate->name = JP2S(pName);
	pTemplate->content = JP2S(pContent);
	pTemplate->parent = JP2S(pParent);

	Storage::instance().updateTemplate(pTemplate->name, pTemplate, driver);
}

void K8SWatchCallback::onTemplateModified(const rapidjson::Value& pDocument)
{
	return onTemplateAdded(pDocument, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onTemplateDeleted(const rapidjson::Value& pDocument)
{
	auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
	assert(pName != nullptr && pName->IsString());
	auto name = JP2S(pName);
	Storage::instance().deleteTemplate(name);
}

void K8SWatchCallback::prePodList()
{
	Storage::instance().preListEndpoint();
}

void K8SWatchCallback::postEndpointList()
{
	Storage::instance().postListEndpoint();
}

void K8SWatchCallback::onEndpointAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver)
{
	auto pTars = rapidjson::GetValueByPointer(pDocument, "/spec/tars");
	if (pTars == nullptr)
	{
		return;
	}
	assert(pTars->IsObject());

	auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
	assert(pName != nullptr && pName->IsString());

	auto pTEndpoint = buildTendpointFromDocument(*pTars);

	if (pTEndpoint == nullptr)
	{
		TLOGERROR("We Got K8S TEndpoint Resource|" << JP2S(pName) << ", But Read Information Error");
		Storage::instance().updateEndpoint(JP2S(pName), pTEndpoint, driver);
		return;
	}

	auto pPods = rapidjson::GetValueByPointer(pDocument, "/status/pods");
	if (pPods != nullptr)
	{
		assert(pPods->IsArray());
		for (const auto& pod: pPods->GetArray())
		{
			auto pPodName = rapidjson::GetValueByPointer(pod, "/name");
			assert(pPodName != nullptr && pPodName->IsString());

			auto pPresentState = rapidjson::GetValueByPointer(pod, "/presentState");
			assert(pPresentState != nullptr && pPresentState->IsString());
			auto presentState = JP2S(pPresentState);

			auto pSettingState = rapidjson::GetValueByPointer(pod, "/settingState");
			assert(pPresentState != nullptr && pPresentState->IsString());
			auto settingState = JP2S(pSettingState);
			if (presentState == "Active" && settingState == "Active")
			{
				pTEndpoint->activatedPods.insert(JP2S(pPodName));
			}else {
                pTEndpoint->inActivatedPods.insert(JP2S(pPodName));
            }
		}
	}

	Storage::instance().updateEndpoint(JP2S(pName), pTEndpoint, driver);
}

void K8SWatchCallback::onEndpointModified(const rapidjson::Value& pDocument)
{
	return onEndpointAdded(pDocument, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onEndpointDeleted(const rapidjson::Value& pDocument)
{
	auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
	assert(pName != nullptr && pName->IsString());
	Storage::instance().deleteTEndpoint(JP2S(pName));
}

void K8SWatchCallback::onConfigAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver)
{
	constexpr char ExpectedConfigName[] = "tars-tarsregistry";
	auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
	assert(pName != nullptr && pName->IsString());
	if (JP2S(pName) != ExpectedConfigName)
	{
		return;
	}
	auto pUpChain = buildUpChainFromDocument(pDocument);
	Storage::instance().updateUpChain(pUpChain);
}

void K8SWatchCallback::onConfigModified(const rapidjson::Value& pDocument)
{
	onConfigAdded(pDocument, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onConfigDeleted(const rapidjson::Value& pDocument)
{
	constexpr char ExpectedConfigName[] = "tars-tarsregistry";
	auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
	assert(pName != nullptr && pName->IsString());
	if (JP2S(pName) != ExpectedConfigName)
	{
		return;
	}
	Storage::instance().updateUpChain(nullptr);
}