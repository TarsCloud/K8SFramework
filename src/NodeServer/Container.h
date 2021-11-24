
#pragma once

#include <string>

namespace container {
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

    bool loadValues();
};
