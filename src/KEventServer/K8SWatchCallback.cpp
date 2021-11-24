
#include "K8SWatchCallback.h"
#include "ESHelper.h"
#include "util/tc_timer.h"
#include <rapidjson/pointer.h>
#include <rapidjson/writer.h>
#include <iostream>

static std::function<std::string()> buildESIndex{};

static std::string getJsonString(const rapidjson::Value& v)
{
    rapidjson::StringBuffer buffer;
    buffer.Clear();
    rapidjson::Writer<rapidjson::StringBuffer> writer(buffer);
    v.Accept(writer);
    return { buffer.GetString(), buffer.GetSize() };
}

static void buildESPostContent(const rapidjson::Value& pDoc, std::ostringstream& stream)
{
    auto pUID = rapidjson::GetValueByPointer(pDoc, "/metadata/uid");
    auto pRV = rapidjson::GetValueByPointer(pDoc, "/metadata/resourceVersion");
    if (pUID == nullptr || pRV == nullptr)
    {
        return;
    }
    assert(pUID->IsString());
    assert(pRV->IsString());
    auto key = std::string{ pUID->GetString(), pUID->GetStringLength() } + ":" + std::string{ pRV->GetString(), pRV->GetStringLength() };

    stream << (R"({"create":{"_id":")") << key << ("\"}}\n");
    rapidjson::Document v;
    v.CopyFrom(pDoc, v.GetAllocator());
    v.AddMember("@timestamp", TNOW, v.GetAllocator());
    v.RemoveMember("metadata");
    stream << getJsonString(v);
    stream << "\n";
}

static void write2ES(const rapidjson::Value& pDoc)
{
    static std::ostringstream stream{};
    static TC_Timer timer{};
    static std::once_flag onceFlag{};
    std::call_once(onceFlag, []()
    {
        timer.startTimer(1);
    });

    buildESPostContent(pDoc, stream);
    auto context = std::make_shared<ESRequestContext>();
    context->uri = std::string("/" + buildESIndex() + "/_bulk");
    context->body = stream.str();
    ESHelper::post2ESWithRetry(&timer, context);
    stream.str("");
}

void K8SWatchCallback::setESIndex(const string& index)
{
    buildESIndex = [index]()
    {
        return index;
    };
}

void K8SWatchCallback::onEventsAdded(const rapidjson::Value& value, K8SWatchEventDrive driver)
{
    write2ES(value);
}

void K8SWatchCallback::onEventsModified(const rapidjson::Value& value)
{
    write2ES(value);
}

void K8SWatchCallback::onEventsAddedWithFilter(const rapidjson::Value& value, K8SWatchEventDrive driver)
{
    auto pInvolvedNamespace = rapidjson::GetValueByPointer(value, "/involvedObject/namespace");
    std::cout << getJsonString(value) << std::endl;
    if (pInvolvedNamespace != nullptr)
    {
        //We Only Record Cluster Level Events Here , If pInvolvedNamespace!=Null, Means It's Not A Cluster Level Event;
        return;
    }
    write2ES(value);
}

void K8SWatchCallback::onEventsModifiedWithFilter(const rapidjson::Value& value)
{
    onEventsAddedWithFilter(value, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onEventsDeleted(const rapidjson::Value& value)
{
}
