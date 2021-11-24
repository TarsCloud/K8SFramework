

#include "Container.h"
#include "servant/RemoteLogger.h"
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

	static inline bool getEnvValue(const char* envKey, std::string& envValue)
	{
		char* env = getenv(envKey);
		if (env == nullptr)
		{
			TLOGERROR("get \"" << envKey << "\" error" << std::endl;);
			std::cerr << "get " << envKey << " error" << std::endl;
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
				getEnvValue(ServerLauncherArgvEnvKey, serverLauncherArgv);
	}
}

