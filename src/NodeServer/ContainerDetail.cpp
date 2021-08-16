

#include "ContainerDetail.h"
#include <cstdlib>
#include <iostream>

namespace container_detail {

    std::string podIP;
    std::string podName;
    std::string listenAddress;
    std::string imageBindServerApp;
    std::string imageBindServerName;
    std::string imageBindServerType;
    std::string imageBindServerBinDir;
    std::string imageBindServerDataDir;
    std::string imageBindServerLogDir;

    std::string imageBindServerConfFile;
    std::string imageBindServerLauncherFile;
    std::string imageBindServerLauncherArgv;

    constexpr char PodNameEnvKey[] = "PodName";
    constexpr char PodIpEnvKey[] = "PodIP";
    constexpr char listenAddressEnvKey[] = "ListenAddress";
    constexpr char ServerAppEnvKey[] = "ServerApp";
    constexpr char ServerNameEnvKey[] = "ServerName";
    constexpr char ServerTypeEnvKey[] = "ServerType";
    constexpr char SeverBinDirEnvKey[] = "ServerBinDir";
    constexpr char SeverDataDirEnvKey[] = "ServerDataDir";
    constexpr char SeverLogDirEnvKey[] = "ServerLogDir";
    constexpr char ServerConfFileEnvKey[] = "ServerConfFile";
    constexpr char ServerLauncherFileEnvKey[] = "ServerLauncherFile";
    constexpr char ServerLauncherArgvEnvKey[] = "ServerLauncherArgv";

    static inline bool getEnvValue(const char *envKey, std::string &envValue) {
        char *env = getenv(envKey);
        if (env == nullptr) {
            std::cerr << "get " << envKey << " error" << std::endl;
            return false;
        }
        envValue = std::string(env);
        return true;
    }

    // static inline bool getEnvValue(const char *envKey, long &envValue) {
    //     char *env = getenv(envKey);
    //     if (env == nullptr) {
    //         std::cerr << "get " << envKey << " error" << std::endl;
    //         return false;
    //     }
    //     envValue = std::strtol(env, nullptr, 10);
    //     return true;
    // }


    bool loadContainerDetailFromEnv() {

        return
                getEnvValue(PodNameEnvKey, podName) &&
                getEnvValue(PodIpEnvKey, podIP) &&
                getEnvValue(listenAddressEnvKey,listenAddress) &&

                getEnvValue(ServerAppEnvKey, imageBindServerApp) &&
                getEnvValue(ServerNameEnvKey, imageBindServerName) &&

                getEnvValue(ServerTypeEnvKey, imageBindServerType) &&
                getEnvValue(SeverBinDirEnvKey, imageBindServerBinDir) &&
                getEnvValue(SeverDataDirEnvKey, imageBindServerDataDir) &&
                getEnvValue(SeverLogDirEnvKey, imageBindServerLogDir) &&
                getEnvValue(ServerConfFileEnvKey, imageBindServerConfFile) &&
                getEnvValue(ServerLauncherFileEnvKey, imageBindServerLauncherFile) &&
                getEnvValue(ServerLauncherArgvEnvKey, imageBindServerLauncherArgv);
    }
}





