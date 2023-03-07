#include "K8SWatchCallback.h"
#include "ESHelper.h"
#include "TCodec.h"
#include "util/tc_timer.h"
#include <iostream>

static std::function<std::string()> buildESIndex{};

static void buildESPostContent(const boost::json::value& value, std::ostringstream& stream)
{
    std::string key;
    try
    {
        std::string uid;
        std::string version;
        READ_FROM_JSON(uid, value.at_pointer("/metadata/uid"));
        READ_FROM_JSON(version, value.at_pointer("/metadata/resourceVersion"));
        key = uid + ":" + version;
    }
    catch (...)
    {
        return;
    }
    stream << (R"({"create":{"_id":")") << key << ("\"}}\n");
    auto object = value.get_object();
    object.erase("metadata");
    object.insert({{ "@timestamp", TNOW }});
    stream << boost::json::serialize(object);
    stream << "\n";
}

static void write2ES(const boost::json::value& pDoc)
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

void K8SWatchCallback::onEventsAdded(const boost::json::value& value, K8SWatchEventDrive driver)
{
    write2ES(value);
}

void K8SWatchCallback::onEventsModified(const boost::json::value& value)
{
    write2ES(value);
}

void K8SWatchCallback::onEventsAddedWithFilter(const boost::json::value& value, K8SWatchEventDrive driver)
{
    boost::json::error_code ec{};
    auto pInvolvedNamespace = value.find_pointer("/involvedObject/namespace", ec);
    if (ec || pInvolvedNamespace == nullptr)
    {
        write2ES(value);
        //We Only Record Cluster Level Events Here , If pInvolvedNamespace!=Null, Means It's Not A Cluster Level Event;
        return;
    }
}

void K8SWatchCallback::onEventsModifiedWithFilter(const boost::json::value& value)
{
    onEventsAddedWithFilter(value, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onEventsDeleted(const boost::json::value& value)
{
}
