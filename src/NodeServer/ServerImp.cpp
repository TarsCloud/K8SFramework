
#include "ServerImp.h"
#include "ServerObject.h"

int ServerImp::keepAlive(const tars::ServerInfo& serverInfo, tars::TarsCurrentPtr current)
{
	ServerObject::keepAlive(serverInfo);
	return 0;
}

int ServerImp::keepActiving(const tars::ServerInfo& serverInfo, tars::TarsCurrentPtr current)
{
	ServerObject::keepActiving(serverInfo);
	return 0;
}

tars::UInt32 ServerImp::getLatestKeepAliveTime(tars::CurrentPtr current)
{
	return TNOW;
}

int ServerImp::reportVersion(const string& app, const string& serverName, const string& version, tars::TarsCurrentPtr current)
{
	return 0;
}
