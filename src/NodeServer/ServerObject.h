#include "servant/NodeF.h"
#include <string>

struct ServerObject
{
	static int startServer(const std::string& application, const std::string& serverName, std::string& result);

	static int stopServer(const std::string& application, const std::string& serverName, std::string& result);

	static int restartServer(const std::string& application, const std::string& serverName, std::string& result);

	static int addFile(const std::string& application, const std::string& serverName, const std::string& file, std::string& result);

	static int notifyServer(const std::string& application, const std::string& serverName, const std::string& command, std::string& result);

	static void keepActiving(const tars::ServerInfo& serverInfo);

	static void keepAlive(const tars::ServerInfo& serverInfo);

	static void startPatrol();
};
