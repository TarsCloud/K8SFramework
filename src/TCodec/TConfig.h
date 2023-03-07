#pragma once

#include "TCodec.h"
#include "K8SParams.h"

struct TConfig
{
    std::string resourceName;
    std::string appName;
    std::string serverName;
    std::string configContent;
    std::string configName;
    std::string podSeq{ "m" };
};

ENCODE_STRUCT_TO_JSON(TConfig, s, j)
{
    j = boost::json::object{
            { "apiVersion",    API_GROUP_VERSION },
            { "kind",          "TConfig" },
            {
              "metadata",      boost::json::object{
                    { "namespace", K8SParams::Namespace() },
                    { "name",      s.resourceName },
            }
            },
            { "app",           s.appName },
            { "server",        s.serverName },
            { "configName",    s.configName },
            { "configContent", s.configContent },
            { "activated",     true },
    };
}
