
#pragma once

#include "K8SWatcher.h"

struct K8SWatchCallback
{
	static void setESIndex(const std::string& index);

	static void onEventsAdded(const boost::json::value& value, K8SWatchEventDrive driver);

	static void onEventsAddedWithFilter(const boost::json::value& value, K8SWatchEventDrive driver);

	static void onEventsModified(const boost::json::value& value);

	static void onEventsModifiedWithFilter(const boost::json::value& value);

	static void onEventsDeleted(const boost::json::value& value);

};