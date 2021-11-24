
#pragma once

#include "K8SWatcher.h"
#include <rapidjson/document.h>

struct K8SWatchCallback
{
    static void setESIndex(const std::string& index);

    static void onEventsAdded(const rapidjson::Value& value, K8SWatchEventDrive driver);

    static void onEventsAddedWithFilter(const rapidjson::Value& value, K8SWatchEventDrive driver);

    static void onEventsModified(const rapidjson::Value& value);

    static void onEventsModifiedWithFilter(const rapidjson::Value& value);

    static void onEventsDeleted(const rapidjson::Value& value);

};