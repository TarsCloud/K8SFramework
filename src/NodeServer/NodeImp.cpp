#include "NodeImp.h"
#include "ServerObject.h"

void NodeImp::initialize()
{

}

int NodeImp::startServer(const std::string& application, const std::string& serverName, string& result,
        TarsCurrentPtr current)
{
    TLOGDEBUG("startServer:" << application << "," << serverName << endl);
    return ServerObject::startServer(application, serverName, result);
}

int NodeImp::stopServer(const std::string& application, const std::string& serverName, string& result,
        TarsCurrentPtr current)
{
    TLOGDEBUG("stopServer:" << application << "," << serverName << endl);
    return ServerObject::stopServer(application, serverName, result);
}

int NodeImp::restartServer(const std::string& application, const std::string& serverName, std::string& result,
        TarsCurrentPtr current)
{
    TLOGDEBUG("restartServer:" << application << "," << serverName << endl);
    return ServerObject::restartServer(application, serverName, result);
}

int NodeImp::notifyServer(const std::string& application, const std::string& serverName, const std::string& sMsg,
        string& result, TarsCurrentPtr current)
{
    TLOGDEBUG("notifyServer:" << application << "," << serverName << endl);
    return ServerObject::notifyServer(application, serverName, sMsg, result);
}

int
NodeImp::addFile(const std::string& application, const std::string& serverName, const std::string& file, string& result,
        TarsCurrentPtr current)
{
    TLOGDEBUG("addFile:" << application << "." << serverName << ":" << file << endl);
    return ServerObject::addFile(application, serverName, file, result);
}
