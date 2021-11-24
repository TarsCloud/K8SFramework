
#pragma once

#include "K8SWatcher.h"

struct K8SWatchCallback
{
	static void prePodList();

	static void postEndpointList();

	static void onEndpointAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver);

	static void onEndpointModified(const rapidjson::Value& pDocument);

	static void onEndpointDeleted(const rapidjson::Value& pDocument);

	static void preTemplateList();

	static void postTemplateList();

	static void onTemplateAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver);

	static void onTemplateModified(const rapidjson::Value& pDocument);

	static void onTemplateDeleted(const rapidjson::Value& pDocument);

	static void onConfigAdded(const rapidjson::Value& pDocument, K8SWatchEventDrive driver);

	static void onConfigModified(const rapidjson::Value& pDocument);

	static void onConfigDeleted(const rapidjson::Value& pDocument);
};