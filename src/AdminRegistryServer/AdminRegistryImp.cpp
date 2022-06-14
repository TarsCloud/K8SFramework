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
#include "Node.h"
#include "util/tc_port.h"

extern TC_Config * g_pconf;

void AdminRegistryImp::initialize()
{
    TLOG_DEBUG("begin AdminRegistryImp init"<<endl);

}

string AdminRegistryImp::nodeNameToDns(const string &nodeName)
{
	string dns = nodeName;
	string::size_type pos = nodeName.rfind("-");
	if(pos != string::npos)
	{
		dns = nodeName + "." + nodeName.substr(0, pos) + "." + TC_Port::getEnv("Namespace");
	}

	TLOG_DEBUG("nodeName:" << nodeName << ", dns:" << dns <<endl);

	return dns;
}

bool AdminRegistryImp::pingNode(const string & name, string & result, tars::CurrentPtr current)
{
    try
    {
        TLOG_DEBUG("into " << __FUNCTION__ << "|" << name << endl);
		NodePrx nodePrx = Application::getCommunicator()->stringToProxy<NodePrx>("tars.tarsnode.NodeObj@tcp -h " + nodeNameToDns(name) + " -p 19385");
		nodePrx->tars_ping();
		result = "ping succ";
		return 0;
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
    TLOG_DEBUG("AdminRegistryImp::startServer: "<< application << "." << serverName << ", " << nodeName
        << ", " << current->getHostName() << ":" << current->getPort() <<endl);

    int iRet = EM_TARS_UNKNOWN_ERR;
    try
    {
		NodePrx nodePrx = Application::getCommunicator()->stringToProxy<NodePrx>("tars.tarsnode.NodeObj@tcp -h " + nodeNameToDns(nodeName) + " -p 19385");
		return nodePrx->startServer(application, serverName, result);
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
    TLOG_DEBUG("AdminRegistryImp::stopServer: "<< application << "." << serverName << ", " << nodeName
        << ", " << current->getHostName() << ":" << current->getPort() <<endl);

    int iRet = EM_TARS_UNKNOWN_ERR;
    try
    {
		NodePrx nodePrx = Application::getCommunicator()->stringToProxy<NodePrx>("tars.tarsnode.NodeObj@tcp -h " + nodeNameToDns(nodeName) + " -p 19385");
		return nodePrx->stopServer(application, serverName, result);
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
    TLOG_DEBUG(" AdminRegistryImp::restartServer: " << application << "." << serverName << "," << nodeName << "," << current->getHostName() << ":" << current->getPort() <<endl);

    int iRet = EM_TARS_SUCCESS;
    try
    {
		NodePrx nodePrx = Application::getCommunicator()->stringToProxy<NodePrx>("tars.tarsnode.NodeObj@tcp -h " + nodeNameToDns(nodeName) + " -p 19385");
		return nodePrx->restartServer(application, serverName, result);
    }
    catch(TarsException & ex)
    {
    
        TLOG_ERROR(("AdminRegistryImp::restartServer '" + application  + "." + serverName + "_" + nodeName
                + "' exception:" + ex.what())<<endl);
        iRet = EM_TARS_UNKNOWN_ERR;
        RemoteNotify::getInstance()->report(string("restart server:") + ex.what(), application, serverName, nodeName);
    }

    return iRet;
}


int AdminRegistryImp::notifyServer(const string & application, const string & serverName, const string & nodeName, const string &command, string &result, tars::CurrentPtr current)
{
    TLOG_DEBUG("AdminRegistryImp::notifyServer: " << application << "." << serverName << "," << nodeName << "," << current->getHostName() << ":" << current->getPort() <<endl);
    int iRet = EM_TARS_UNKNOWN_ERR;
    try
    {
		NodePrx nodePrx = Application::getCommunicator()->stringToProxy<NodePrx>("tars.tarsnode.NodeObj@tcp -h " + nodeNameToDns(nodeName) + " -p 19385");
		return nodePrx->notifyServer(application, serverName, command, result);
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

class PluginHttpCallback : public TC_HttpAsync::RequestCallback
{
public:
	PluginHttpCallback(const CurrentPtr &current) : _current(current)
	{
	}
	virtual void onSucc(TC_HttpResponse &stHttpResponse)
	{
		_rsp = stHttpResponse;

		TLOG_DEBUG(_rsp.getStatus() << ", " << _rsp.getContent() << endl);

		AdminReg::async_response_registerPlugin(_current, _rsp.getStatus() == 200?0:-1);
	}
	virtual void onFailed(FAILED_CODE ret, const string &info)
	{
		TLOG_ERROR("onFailed, code:" << ret << ", info:" << info << endl);
		AdminReg::async_response_registerPlugin(_current, -1);
	}

	virtual void onClose()
	{
//		LOG_CONSOLE_DEBUG << "onClose:" << _sUrl << endl;
	}
public:
	CurrentPtr _current;
	TC_HttpResponse _rsp;
};


int AdminRegistryImp::registerPlugin(const PluginConf &conf, CurrentPtr current)
{
	try
	{
		TC_HttpRequest stHttpReq;
		stHttpReq.setHeader("Connection", "Close");

		string url ="http://tars-tarsweb:3000/pages/plugin/api/install";

		stHttpReq.setPostRequest(url, conf.writeToJsonString(), true);

		TLOG_DEBUG("name: " << conf.name << ", obj:" << conf.obj << ", url:" << url << endl);

		current->sendResponse(false);

		PluginHttpCallback *callback = new PluginHttpCallback(current);

		TC_HttpAsync::RequestCallbackPtr p(callback);

		g_app._ast.doAsyncRequest(stHttpReq, p);

		return 0;
	}
	catch (exception & ex)
	{
		TLOG_ERROR(ex.what() << endl);
		return EM_TARS_UNKNOWN_ERR;
	}

	return -1;
}


class AuthHttpCallback : public TC_HttpAsync::RequestCallback
{
public:
	AuthHttpCallback(const string &application, const string &serverName, const string &role, const CurrentPtr &current) : _application(application), _serverName(serverName), _role(role), _current(current)
	{
	}

	void onDev(const AuthConfList &acl)
	{

		bool has = false;
		for(auto flag: acl.auths)
		{
			if (flag.role == "admin")
			{
				has = true;
				break;
			}

			if (flag.flag == "*" && (flag.role == "developer" || flag.role == "operator"))
			{
				has = true;
				break;
			}

			if ((flag.flag == _application || flag.flag == (_application + ".*")) &&
				(flag.role == "developer" || flag.role == "operator"))
			{
				has = true;
				break;
			}

			if (!_serverName.empty())
			{
				if (flag.flag == (_application + "." + _serverName) &&
					(flag.role == "developer" || flag.role == "operator"))
				{
					has = true;
					break;
				}
			}
		}

		AdminReg::async_response_hasDevAuth(_current, 0, has);
	}

	void onOpe(const AuthConfList &acl)
	{

		bool has = false;
		for(auto flag: acl.auths)
		{
			if (flag.role == "admin")
			{
				has = true;
				break;
			}

			if (flag.flag == "*" && (flag.role == "operator"))
			{
				has = true;
				break;
			}

			if ((flag.flag == _application || flag.flag == (_application + ".*")) &&
				(flag.role == "operator"))
			{
				has = true;
				break;
			}

			if (!_serverName.empty())
			{
				if (flag.flag == (_application + "." + _serverName) &&
					(flag.role == "operator"))
				{
					has = true;
					break;
				}
			}
		}

		AdminReg::async_response_hasOpeAuth(_current, 0, has);
	}

	void onAdmin(const AuthConfList &acl)
	{
		bool has = false;
		for(auto flag: acl.auths)
		{
			if (flag.role == "admin")
			{
				has = true;
				break;
			}
		}

		AdminReg::async_response_hasAdminAuth(_current, 0, has);
	}

	virtual void onSucc(TC_HttpResponse &stHttpResponse)
	{
		_rsp = stHttpResponse;

		TLOG_DEBUG(_rsp.getStatus() << ", " << _rsp.getContent() << endl);

		if(_rsp.getStatus() == 200)
		{

			AuthConfList acl;
			acl.readFromJsonString(_rsp.getContent());

			if (_role == "developer")
			{
				onDev(acl);
			}
			else if (_role == "operator")
			{
				onOpe(acl);
			}
			else
			{
				onAdmin(acl);
			}
		}
		else
		{
			if (_role == "developer")
			{
				AdminReg::async_response_hasDevAuth(_current, -1, false);
			}
			else if (_role == "operator")
			{
				AdminReg::async_response_hasOpeAuth(_current, -1, false);
			}
			else
			{
				AdminReg::async_response_hasAdminAuth(_current, -1, false);
			}
		}
	}

	virtual void onFailed(FAILED_CODE ret, const string &info)
	{
		TLOG_ERROR("onFailed, code:" << ret << ", info:" << info << endl);
		if (_role == "developer")
		{
			AdminReg::async_response_hasDevAuth(_current, -1, false);
		}
		else if (_role == "operator")
		{
			AdminReg::async_response_hasOpeAuth(_current, -1, false);
		}
		else
		{
			AdminReg::async_response_hasAdminAuth(_current, -1, false);
		}
	}

	virtual void onClose()
	{
//		LOG_CONSOLE_DEBUG << "onClose:" << _sUrl << endl;
	}
public:
	string _application;
	string _serverName;
	string _role;
	CurrentPtr _current;
	TC_HttpResponse _rsp;
};

int AdminRegistryImp::hasDevAuth(const string &application, const string & serverName, const string & uid, bool &has, CurrentPtr current)
{
	try
	{
		TC_HttpRequest stHttpReq;
		stHttpReq.setHeader("Connection", "Close");

		string url = "http://tars-tarsweb:3000/pages/server/api/authList?uid=" + uid;
		stHttpReq.setGetRequest(url, true);

		TLOG_DEBUG("application: " << application << ", serverName:" << serverName << ", uid:" << uid << ", url:" << url << endl);

		current->sendResponse(false);

		AuthHttpCallback *callback = new AuthHttpCallback(application, serverName, "developer", current);

		TC_HttpAsync::RequestCallbackPtr p(callback);

		g_app._ast.doAsyncRequest(stHttpReq, p);

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
	try
	{
		TC_HttpRequest stHttpReq;
		stHttpReq.setHeader("Connection", "Close");

		string url = "http://tars-tarsweb:3000/pages/server/api/authList?uid=" + uid;
		stHttpReq.setGetRequest(url, true);

		TLOG_DEBUG("application: " << application << ", serverName:" << serverName << ", uid:" << uid << ", url:" << url << endl);

		current->sendResponse(false);

		AuthHttpCallback *callback = new AuthHttpCallback(application, serverName, "operator", current);

		TC_HttpAsync::RequestCallbackPtr p(callback);

		g_app._ast.doAsyncRequest(stHttpReq, p);

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
	try
	{
		TC_HttpRequest stHttpReq;
		stHttpReq.setHeader("Connection", "Close");

		string url = "http://tars-tarsweb:3000/pages/server/api/authList?uid=" + uid;
		stHttpReq.setGetRequest(url, true);

		TLOG_DEBUG("uid:" << uid << ", url:" << url << endl);

		current->sendResponse(false);

		AuthHttpCallback *callback = new AuthHttpCallback("", "", "admin", current);

		TC_HttpAsync::RequestCallbackPtr p(callback);

		g_app._ast.doAsyncRequest(stHttpReq, p);

		return 0;
	}
	catch (exception & ex)
	{
		TLOG_ERROR(ex.what() << endl);
		return EM_TARS_UNKNOWN_ERR;
	}

	return -1;
}


class TicketHttpCallback : public TC_HttpAsync::RequestCallback
{
public:
	TicketHttpCallback(const CurrentPtr &current) : _current(current)
	{
	}
	virtual void onSucc(TC_HttpResponse &stHttpResponse)
	{
		_rsp = stHttpResponse;

		TLOG_DEBUG(_rsp.getStatus() << ", " << _rsp.getContent() << endl);

		JsonValueObjPtr oPtr = JsonValueObjPtr::dynamicCast(TC_Json::getValue(_rsp.getContent()));

		JsonValueStringPtr sPtr = JsonValueStringPtr::dynamicCast(oPtr->value["uid"]);

		string uid = sPtr->value;

		AdminReg::async_response_checkTicket(_current, 0, uid);
	}

	virtual void onFailed(FAILED_CODE ret, const string &info)
	{
		TLOG_ERROR("onFailed, code:" << ret << ", info:" << info << endl);
		AdminReg::async_response_registerPlugin(_current, -1);
	}

	virtual void onClose()
	{
//		LOG_CONSOLE_DEBUG << "onClose:" << _sUrl << endl;
	}
public:
	CurrentPtr _current;
	TC_HttpResponse _rsp;
};

int AdminRegistryImp::checkTicket(const string & ticket, string &uid, CurrentPtr current)
{
	try
	{
		TC_HttpRequest stHttpReq;
		stHttpReq.setHeader("Connection", "Close");

		string url = "http://tars-tarsweb:3000/pages/server/api/ticket";

		stHttpReq.setGetRequest(url, true);
		TLOG_DEBUG("ticket:" << ticket << ", url:" << url << endl);

		current->sendResponse(false);

		TicketHttpCallback *callback = new TicketHttpCallback( current);

		TC_HttpAsync::RequestCallbackPtr p(callback);

		g_app._ast.doAsyncRequest(stHttpReq, p);


		return 0;
	}
	catch (exception & ex)
	{
		TLOG_ERROR(ex.what() << endl);
		return EM_TARS_UNKNOWN_ERR;
	}

	return -1;
}
