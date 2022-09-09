
#include "K8SWatchCallback.h"
#include "Storage.h"
#include <servant/RemoteLogger.h>
#include <rapidjson/pointer.h>
#include <rapidjson/stringbuffer.h>
#include <rapidjson/writer.h>
#include <rapidjson/document.h>
#include <iostream>

static inline std::string JP2S(const rapidjson::Value* p)
{
    assert(p != nullptr && p->IsString());
    return { p->GetString(), p->GetStringLength() };
}

static void PrintJsonText(const rapidjson::Value& v)
{
    rapidjson::StringBuffer buffer;

    buffer.Clear();

    rapidjson::Writer <rapidjson::StringBuffer> writer(buffer);
    v.Accept(writer);

    std::cout << buffer.GetString() << std::endl;
}

static std::shared_ptr <TEndpoint> buildTE(const rapidjson::Value& pDocument)
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

        pTEndpoint->tAdapters.push_back(pAdapter);
    }
    return pTEndpoint;
}


static std::shared_ptr <UPChain> buildUPChain(const rapidjson::Value& pDocument)
{
    auto pUPChain = rapidjson::GetValueByPointer(pDocument, "/upChain");
    if (pUPChain == nullptr)
    {
        return nullptr;
    }

    if (!pUPChain->IsObject())
    {
        return nullptr;
    }

    auto result = std::make_shared<UPChain>();

    for (auto&& item: pUPChain->GetObject())
    {
        std::vector <tars::EndpointF> epv;

        auto&& value = item.value;
        if (!value.IsArray())
        {
            continue;
        }

        auto&& array = value.GetArray();
        epv.reserve(array.Size());

        for (auto& ep: array)
        {
            auto&& hostRef = ep["host"];
            if (!hostRef.IsString())
            {
                return nullptr;
            }

            auto&& portRef = ep["port"];
            if (!portRef.IsInt())
            {
                return nullptr;
            }

            auto&& timeoutRef = ep["timeout"];
            if (!portRef.IsInt())
            {
                return nullptr;
            }

            tars::EndpointF f{};
            f.host = std::string(hostRef.GetString(), hostRef.GetStringLength());
            f.port = portRef.GetInt();
            f.timeout = timeoutRef.GetInt();

            auto pIsTcp = rapidjson::GetValueByPointer(ep, "/isTcp");
            if (pIsTcp == nullptr || !pIsTcp->IsBool() || pIsTcp->GetBool())
            {
                f.istcp = 1;
            }

            epv.emplace_back(f);
        }

        constexpr char DEFAULT_UPCHAIN_NAME[] = "default";
        constexpr size_t DEFAULT_UPCHAIN_NAME_LEN = sizeof(DEFAULT_UPCHAIN_NAME) - 1;

        auto&& key = item.name;
        assert(key.IsString());
        auto name = key.GetString();
        auto nameLen = key.GetStringLength();

        if (nameLen != DEFAULT_UPCHAIN_NAME_LEN || strncmp(name, DEFAULT_UPCHAIN_NAME, DEFAULT_UPCHAIN_NAME_LEN) != 0)
        {
            result->customs[std::string(name, nameLen)] = std::move(epv);
        }
        else
        {
            result->defaults = std::move(epv);
        }
    }
    return result;
}

void K8SWatchCallback::preTTList()
{
    Storage::instance().preListTemplate();
}

void K8SWatchCallback::postTTList()
{
    Storage::instance().postListTemplate();
}

void K8SWatchCallback::onTTAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver)
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

void K8SWatchCallback::onTTModified(const rapidjson::Value& pDocument)
{
    return onTTAdded(pDocument, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onTTDeleted(const rapidjson::Value& pDocument)
{
    auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(pName != nullptr && pName->IsString());
    auto name = JP2S(pName);
    Storage::instance().deleteTemplate(name);
}

void K8SWatchCallback::preTEList()
{
    Storage::instance().preListEndpoint();
}

void K8SWatchCallback::postTEList()
{
    Storage::instance().postListEndpoint();
}

void K8SWatchCallback::onTEAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver)
{
    auto pTars = rapidjson::GetValueByPointer(pDocument, "/spec/tars");
    if (pTars == nullptr)
    {
        return;
    }
    assert(pTars->IsObject());

    auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(pName != nullptr && pName->IsString());

    auto pTEndpoint = buildTE(*pTars);

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
            }
            else
            {
                pTEndpoint->inActivatedPods.insert(JP2S(pPodName));
            }
        }
    }

    Storage::instance().updateEndpoint(JP2S(pName), pTEndpoint, driver);
}

void K8SWatchCallback::onTEModified(const rapidjson::Value& pDocument)
{
    return onTEAdded(pDocument, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onTEDeleted(const rapidjson::Value& pDocument)
{
    auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(pName != nullptr && pName->IsString());
    Storage::instance().deleteTEndpoint(JP2S(pName));
}

void K8SWatchCallback::onTFCAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver)
{
    constexpr char ExpectedName[] = "tars-framework";
    auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(pName != nullptr && pName->IsString());
    if (JP2S(pName) != ExpectedName)
    {
        return;
    }
    auto pUpChain = buildUPChain(pDocument);
    Storage::instance().updateUpChain(pUpChain);
}

void K8SWatchCallback::onTFCModified(const rapidjson::Value& pDocument)
{
    onTFCAdded(pDocument, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onTFCDeleted(const rapidjson::Value& pDocument)
{
    constexpr char ExpectedName[] = "tars-framework";
    auto pName = rapidjson::GetValueByPointer(pDocument, "/metadata/name");
    assert(pName != nullptr && pName->IsString());
    if (JP2S(pName) != ExpectedName)
    {
        return;
    }
    Storage::instance().updateUpChain(nullptr);
}
