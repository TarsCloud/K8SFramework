#include "NodeImp.h"
#include "NodeServer.h"
#include "ServerObject.h"

void NodeImp::initialize()
{
}

int NodeImp::startServer(const string& application, const string& serverName, string& result, TarsCurrentPtr current)
{
	TLOGDEBUG("startServer:" << application << "," << serverName << endl);
	return ServerObject::startServer(application, serverName, result);
}

int NodeImp::stopServer(const string& application, const string& serverName, string& result, TarsCurrentPtr current)
{
	TLOGDEBUG("stopServer:" << application << "," << serverName << endl);
	return ServerObject::stopServer(application, serverName, result);
}

int NodeImp::restartServer(const std::string& application, const std::string& serverName, std::string& result, tars::TarsCurrentPtr current)
{
	TLOGDEBUG("restartServer:" << application << "," << serverName << endl);
	return ServerObject::restartServer(application, serverName, result);
}

int NodeImp::notifyServer(const string& application, const string& serverName, const string& sMsg, string& result, TarsCurrentPtr current)
{
	TLOGDEBUG("notifyServer:" << application << "," << serverName << endl);
	return ServerObject::notifyServer(application, serverName, sMsg, result);
}

int NodeImp::addFile(const string& application, const string& serverName, const string& file, string& result, TarsCurrentPtr current)
{
	TLOGDEBUG("addFile:" << application << "." << serverName << ":" << file << endl);
	return ServerObject::addFile(application, serverName, file, result);
}
