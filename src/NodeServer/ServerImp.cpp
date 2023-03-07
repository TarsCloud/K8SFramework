
#include "ServerImp.h"
#include "ServerObject.h"

int ServerImp::keepAlive(const tars::ServerInfo& serverInfo, TarsCurrentPtr current)
{
    ServerObject::keepAlive(serverInfo);
    return 0;
}

int ServerImp::keepActiving(const tars::ServerInfo& serverInfo, TarsCurrentPtr current)
{
    ServerObject::keepActiving(serverInfo);
    return 0;
}

tars::UInt32 ServerImp::getLatestKeepAliveTime(tars::CurrentPtr current)
{
    return TNOW;
}

int ServerImp::reportVersion(const std::string& app, const std::string& serverName, const std::string& version,
        TarsCurrentPtr current)
{
    return 0;
}
