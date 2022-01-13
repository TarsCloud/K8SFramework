#include "Container.h"
#include <cstdlib>
#include <iostream>

namespace container
{
    std::string podIP;
    std::string podName;
    std::string listenAddress;
    std::string serverApp;
    std::string serverName;
    std::string serverType;
    std::string serverBinDir;
    std::string serverDataDir;
    std::string serverLogDir;

    std::string serverConfFile;
    std::string serverLauncherFile;
    std::string serverLauncherArgv;
    std::string serverLauncherType;

    static inline bool getEnvValue(const char* envKey, std::string& envValue)
    {
        char* env = getenv(envKey);
        if (env == nullptr)
        {
            std::cout << "get " << envKey << " error" << std::endl;
            return false;
        }
        envValue = std::string(env);
        return true;
    }

    bool loadValues()
    {
        return
                getEnvValue(PodNameEnvKey, podName) &&
                getEnvValue(PodIpEnvKey, podIP) &&
                getEnvValue(listenAddressEnvKey, listenAddress) &&

                getEnvValue(ServerAppEnvKey, serverApp) &&
                getEnvValue(ServerNameEnvKey, serverName) &&

                getEnvValue(ServerTypeEnvKey, serverType) &&
                getEnvValue(SeverBinDirEnvKey, serverBinDir) &&
                getEnvValue(SeverDataDirEnvKey, serverDataDir) &&
                getEnvValue(SeverLogDirEnvKey, serverLogDir) &&
                getEnvValue(ServerConfFileEnvKey, serverConfFile) &&
                getEnvValue(ServerLauncherFileEnvKey, serverLauncherFile) &&
                getEnvValue(ServerLauncherArgvEnvKey, serverLauncherArgv) &&
                getEnvValue(ServerLauncherTypeEnvKey, serverLauncherType);
    }
}

