
#pragma once

#include <string>

namespace container_detail {
    extern std::string podName;
    extern std::string podIP;
    extern std::string listenAddress;
    extern std::string imageBindServerApp;

    extern std::string imageBindServerName;
    extern std::string imageBindServerType;
    extern std::string imageBindServerBinDir;
    extern std::string imageBindServerDataDir;
    extern std::string imageBindServerLogDir;
    extern std::string imageBindServerConfFile;
    extern std::string imageBindServerLauncherFile;
    extern std::string imageBindServerLauncherArgv;

    bool loadContainerDetailFromEnv();
};
