
#pragma once

#include "K8SWatcher.h"

struct K8SWatchCallback
{
    static void preTEList();

    static void postTEList();

    static void onTEAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver);

    static void onTEModified(const rapidjson::Value& pDocument);

    static void onTEDeleted(const rapidjson::Value& pDocument);

    static void preTTList();

    static void postTTList();

    static void onTTAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver);

    static void onTTModified(const rapidjson::Value& pDocument);

    static void onTTDeleted(const rapidjson::Value& pDocument);

    static void onTFCAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver);

    static void onTFCModified(const rapidjson::Value& pDocument);

    static void onTFCDeleted(const rapidjson::Value& pDocument);
};