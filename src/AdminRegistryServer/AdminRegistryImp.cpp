/**
 * Tencent is pleased to support the open source community by making Tars available.
 *
 * Copyright (C) 2016THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License"); you may not use this file except 
 * in compliance with the License. You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed 
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR 
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the 
 * specific language governing permissions and limitations under the License.
 */

#include "AdminRegistryImp.h"
#include "servant/Application.h"
#include "servant/RemoteNotify.h"
#include "AdminRegistryServer.h"
#include "NodeManager.h"
#include "NodePush.h"

extern TC_Config * g_pconf;

void AdminRegistryImp::initialize()
{
    TLOG_DEBUG("begin AdminRegistryImp init"<<endl);

	// _registryPrx = Application::getCommunicator()->stringToProxy<RegistryPrx>(g_pconf->get("/tars/objname<RegistryObjName>"), "tars.tarsregistry.RegistryObj");

	// _registryPrx->tars_async_timeout(60*1000);

}

int AdminRegistryImp::doClose(CurrentPtr current)
{
	TLOG_DEBUG("uid:" << current->getUId() << endl);
	NodeManager::getInstance()->eraseNodeCurrent(current);
	return 0;
}

int AdminRegistryImp::reportNode(const ReportNode &rn, CurrentPtr current)
{
//	TLOG_DEBUG("nodeName:" << nodeName << ", uid:" << current->getUId() << endl);

	NodeManager::getInstance()->createNodeCurrent(rn.nodeName, rn.sid, current);
	return 0;
}

int AdminRegistryImp::deleteNode(const ReportNode &rn, CurrentPtr current)
{
	NodeManager::getInstance()->deleteNodeCurrent(rn.nodeName, rn.sid, current);
	return 0;
}

int AdminRegistryImp::reportResult(int requestId, const string &funcName, int ret, const string &result, CurrentPtr current)
{
	TLOG_DEBUG("requestId:" << requestId << ", " << funcName << ", ret:" << ret << endl);

	NodeManager::getInstance()->reportResult(requestId, funcName, ret, result, current);
	return 0;
}

bool AdminRegistryImp::pingNode(const string & name, string & result, tars::CurrentPtr current)
{
    try
    {
        TLOG_DEBUG("into " << __FUNCTION__ << "|" << name << endl);

		return NodeManager::getInstance()->pingNode(name, result, current);
    }
    catch(TarsException & ex)
    {
        result = string(string(__FUNCTION__)) + " '" + name + "' exception:" + ex.what();
        TLOG_ERROR(result << endl);
        return false;
    }

    return false;
}

int AdminRegistryImp::startServer(const string & application, const string & serverName, const string & nodeName, string & result, tars::CurrentPtr current)
{
    TLOG_DEBUG("AdminRegistryImp::startServer: "<< application << "." << serverName << "_" << nodeName
        << "|" << current->getHostName() << ":" << current->getPort() <<endl);

    int iRet = EM_TARS_UNKNOWN_ERR;
    try
    {
		return NodeManager::getInstance()->startServer(application, serverName, nodeName, result, current);
    }
    catch(TarsException & ex)
    {
        current->setResponse(true);
        result = string(__FUNCTION__) + " '" + application  + "." + serverName + "_" + nodeName
                 + "' TarsException:" + ex.what();
		RemoteNotify::getInstance()->report(string("start server:") + ex.what(), application, serverName, nodeName);
		 
        TLOG_ERROR(result << endl);
    }

	if(iRet != EM_TARS_SUCCESS)
	{
		RemoteNotify::getInstance()->report(string("start server error:" + etos((tarsErrCode)iRet)) , application, serverName, nodeName);
	}
    return iRet;
}


int AdminRegistryImp::stopServer(const string & application, const string & serverName, const string & nodeName, string & result, tars::CurrentPtr current)
{
    TLOG_DEBUG("AdminRegistryImp::stopServer: "<< application << "." << serverName << "_" << nodeName
        << "|" << current->getHostName() << ":" << current->getPort() <<endl);

    int iRet = EM_TARS_UNKNOWN_ERR;
    try
    {
		iRet = NodeManager::getInstance()->startServer(application, serverName, nodeName, result, current);
    }
    catch(TarsException & ex)
    {
        current->setResponse(true);
        result = string(__FUNCTION__) + " '" + application  + "." + serverName + "_" + nodeName
                 + "' Exception:" + ex.what();
		RemoteNotify::getInstance()->report(result, application, serverName, nodeName);
        TLOG_ERROR(result << endl);
    }
    return iRet;
}

int AdminRegistryImp::restartServer(const string & application, const string & serverName, const string & nodeName, string & result, tars::CurrentPtr current)
{
    TLOG_DEBUG(" AdminRegistryImp::restartServer: " << application << "." << serverName << "_" << nodeName << "|" << current->getHostName() << ":" << current->getPort() <<endl);

    int iRet = EM_TARS_SUCCESS;
    try
    {
		
		iRet = NodeManager::getInstance()->stopServer(application, serverName, nodeName, result, current);
    }
    catch(TarsException & ex)
    {
    
        TLOG_ERROR(("AdminRegistryImp::restartServer '" + application  + "." + serverName + "_" + nodeName
                + "' exception:" + ex.what())<<endl);
        iRet = EM_TARS_UNKNOWN_ERR;
        RemoteNotify::getInstance()->report(string("restart server:") + ex.what(), application, serverName, nodeName);
    }

    if(iRet == EM_TARS_SUCCESS)
    {
        try
        {
			return NodeManager::getInstance()->startServer(application, serverName, nodeName, result, current);
        }
        catch (TarsException & ex)
        {
			RemoteNotify::getInstance()->report(string("restart server:") + ex.what(), application, serverName, nodeName);

            result += string(__FUNCTION__) + " '" + application  + "." + serverName + "_" + nodeName
                      + "' Exception:" + ex.what();
            iRet = EM_TARS_UNKNOWN_ERR;
        }
        TLOG_ERROR( result << endl);
    }

    return iRet;
}


int AdminRegistryImp::notifyServer(const string & application, const string & serverName, const string & nodeName, const string &command, string &result, tars::CurrentPtr current)
{
    TLOG_DEBUG("AdminRegistryImp::notifyServer: " << application << "." << serverName << "_" << nodeName << "|" << current->getHostName() << ":" << current->getPort() <<endl);
    int iRet = EM_TARS_UNKNOWN_ERR;
    try
    {
		return NodeManager::getInstance()->notifyServer(application, serverName, nodeName, command, result, current);
    }
    catch(TarsException & ex)
    {
        current->setResponse(true);
        TLOG_ERROR("AdminRegistryImp::notifyServer '"<<(application  + "." + serverName + "_" + nodeName
                + "' Exception:" + ex.what())<<endl);
        RemoteNotify::getInstance()->report(string("notify server:") + ex.what(), application, serverName, nodeName);
    }
    return iRet;
}

int AdminRegistryImp::registerPlugin(const PluginConf &conf, CurrentPtr current)
{
	TLOG_DEBUG("name: " << conf.name << ", obj:" << conf.obj << endl);

	try
	{
		// int ret = DBPROXY->registerPlugin(conf);

		// if(ret < 0)
		// {
		// 	return -1;
		// }

		return 0;
	}
	catch (exception & ex)
	{
		TLOG_ERROR(ex.what() << endl);
		return EM_TARS_UNKNOWN_ERR;
	}

	return -1;
}

int AdminRegistryImp::hasDevAuth(const string &application, const string & serverName, const string & uid, bool &has, CurrentPtr current)
{
	TLOG_DEBUG("application: " << application << ", serverName:" << serverName << ", uid:" << uid << endl);

	try
	{
		// vector<DbProxy::UserFlag> flags;

		// int ret = DBPROXY->getAuth(uid, flags);

		// if(ret < 0)
		// {
		// 	return -1;
		// }

		// has = false;
		// for(auto flag: flags)
		// {
		// 	if(flag.role == "admin")
		// 	{
		// 		has = true;
		// 		return 0;
		// 	}

		// 	if(flag.flag == "*" && (flag.role == "developer" || flag.role == "operator"))
		// 	{
		// 		has = true;
		// 		return 0;
		// 	}

		// 	if((flag.flag == application || flag.flag == (application + ".*")) && (flag.role == "developer" || flag.role == "operator"))
		// 	{
		// 		has = true;
		// 		return 0;
		// 	}

		// 	if(!serverName.empty())
		// 	{
		// 		if (flag.flag == (application + "." + serverName) &&
		// 			(flag.role == "developer" || flag.role == "operator"))
		// 		{
		// 			has = true;
		// 			return 0;
		// 		}
		// 	}
		// }

		return 0;
	}
	catch (exception & ex)
	{
		TLOG_ERROR(ex.what() << endl);
		return EM_TARS_UNKNOWN_ERR;
	}

	return -1;
}

int AdminRegistryImp::hasOpeAuth(const string & application, const string & serverName, const string & uid, bool &has, CurrentPtr current)
{
	TLOG_DEBUG("application: " << application << ", serverName:" << serverName << ", uid:" << uid << endl);

	try
	{
		// vector<DbProxy::UserFlag> flags;

		// int ret = DBPROXY->getAuth(uid, flags);

		// if(ret < 0)
		// {
		// 	return -1;
		// }

		// has = false;
		// for(auto flag: flags)
		// {
		// 	if(flag.role == "admin")
		// 	{
		// 		has = true;
		// 		return 0;
		// 	}

		// 	if(flag.flag == "*" && (flag.role == "operator"))
		// 	{
		// 		has = true;
		// 		return 0;
		// 	}

		// 	if((flag.flag == application || flag.flag == (application + ".*")) && (flag.role == "operator"))
		// 	{
		// 		has = true;
		// 		return 0;
		// 	}

		// 	if(!serverName.empty())
		// 	{
		// 		if (flag.flag == (application + "." + serverName) &&
		// 			(flag.role == "operator"))
		// 		{
		// 			has = true;
		// 			return 0;
		// 		}
		// 	}
		// }

		return 0;
	}
	catch (exception & ex)
	{
		TLOG_ERROR(ex.what() << endl);
		return EM_TARS_UNKNOWN_ERR;
	}

	return -1;
}

int AdminRegistryImp::hasAdminAuth(const string & uid, bool &has, CurrentPtr current)
{
	TLOG_DEBUG(", uid:" << uid << endl);

	try
	{
		// vector<DbProxy::UserFlag> flags;

		// int ret = DBPROXY->getAuth(uid, flags);

		// if(ret < 0)
		// {
		// 	return -1;
		// }

		// has = false;
		// for(auto flag: flags)
		// {
		// 	if(flag.role == "admin")
		// 	{
		// 		has = true;
		// 		return 0;
		// 	}
		// }

		return 0;
	}
	catch (exception & ex)
	{
		TLOG_ERROR(ex.what() << endl);
		return EM_TARS_UNKNOWN_ERR;
	}

	return -1;
}

int AdminRegistryImp::checkTicket(const string & ticket, string &uid, CurrentPtr current)
{
	TLOG_DEBUG("ticket:" << ticket << endl);

	try
	{
		// int ret = DBPROXY->getTicket(ticket, uid);

		// TLOG_DEBUG("ticket:" << ticket << ", ret:" << ret << ", uid:" << uid << endl);

		// if(ret < 0)
		// {
		// 	return -1;
		// }

		return 0;
	}
	catch (exception & ex)
	{
		TLOG_ERROR(ex.what() << endl);
		return EM_TARS_UNKNOWN_ERR;
	}

	return -1;
}
