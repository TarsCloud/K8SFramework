
#pragma once

#include <string>

namespace container
{
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
    constexpr char ServerLauncherTypeEnvKey[] = "ServerLauncherType";

    extern std::string podName;
    extern std::string podIP;
    extern std::string listenAddress;
    extern std::string serverApp;

    extern std::string serverName;
    extern std::string serverType;
    extern std::string serverBinDir;
    extern std::string serverDataDir;
    extern std::string serverLogDir;
    extern std::string serverConfFile;
    extern std::string serverLauncherFile;
    extern std::string serverLauncherArgv;
    extern std::string serverLauncherType;

    bool loadValues();
}
