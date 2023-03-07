
#pragma once

#include "K8SWatcher.h"

struct K8SWatchCallback
{
    static void preTTList();

    static void postTTList();

    static void onTTAdded(const boost::json::value& value, K8SWatchEventDrive driver);

    static void onTTModified(const boost::json::value& value);

    static void onTTDeleted(const boost::json::value& value);


    static void preTEList();

    static void postTEList();

    static void onTEAdded(const boost::json::value& value, K8SWatchEventDrive driver);

    static void onTEModified(const boost::json::value& value);

    static void onTEDeleted(const boost::json::value& value);


    static void onTFCAdded(const boost::json::value& value, K8SWatchEventDrive driver);

    static void onTFCModified(const boost::json::value& value);

    static void onTFCDeleted(const boost::json::value& value);

};
