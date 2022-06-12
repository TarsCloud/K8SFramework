//
// Created by jarod on 2022/5/7.
//

#include "ServerManager.h"
#include "servant/Application.h"
#include "NodePush.h"
#include "ServerObject.h"
#include "NodeServer.h"
#include "Container.h"

// extern BatchPatch * g_BatchPatchThread;

class AdminNodePushPrxCallback : public NodePushPrxCallback
{
public:
	AdminNodePushPrxCallback(AdminRegPrx &prx) : _adminPrx(prx)
	{
	}

	virtual void callback_notifyServer(tars::Int32 requestId, const string &nodeName,  const std::string& application,  const std::string& serverName,  const std::string& command)
	{
		string result;

		int ret = ServerManager::getInstance()->notifyServer(application, serverName, command, result);

		TLOG_DEBUG( "requestId:" << requestId << ", " << command << ", " << result << endl);

		_adminPrx->async_reportResult(NULL, requestId, __FUNCTION__, ret, result);
	}

	virtual void callback_ping(tars::Int32 requestId, const string &nodeName)
	{
		TLOG_DEBUG("requestId:" << requestId << endl);
		_adminPrx->async_reportResult(NULL, requestId, __FUNCTION__, 0, "ping succ");
	}

	virtual void callback_startServer(tars::Int32 requestId, const string &nodeName,  const std::string& application,  const std::string& serverName)
	{
		string result;

		int ret = ServerManager::getInstance()->startServer(application, serverName, result);

		TLOG_DEBUG( result << endl);

		_adminPrx->async_reportResult(NULL, requestId, __FUNCTION__, ret, result);
	}

	virtual void callback_stopServer(tars::Int32 requestId, const string &nodeName,  const std::string& application,  const std::string& serverName)
	{
		string result;

		int ret = ServerManager::getInstance()->stopServer(application, serverName, result);

		TLOG_DEBUG( result << endl);

		_adminPrx->async_reportResult(NULL, requestId, __FUNCTION__, ret, result);
	}

protected:
	AdminRegPrx _adminPrx;
};

void ServerManager::terminate()
{
	std::lock_guard<std::mutex> lock(_mutex);
	_terminate = true;
	_cond.notify_one();
}

void ServerManager::initialize(const string &adminObj)
{
	_adminObj = adminObj;

	createAdminPrx();

	start();
}

void ServerManager::createAdminPrx()
{
	AdminRegPrx prx = Application::getCommunicator()->stringToProxy<AdminRegPrx>(_adminObj);

	NodePushPrxCallbackPtr callback = new AdminNodePushPrxCallback(prx);

	prx->tars_set_push_callback(callback);

	_adminPrx = prx;
}

void ServerManager::run()
{
	ReportNode rn;
	// PlatformInfo platformInfo;
	rn.nodeName = TC_Port::getEnv(container::PodNameEnvKey);
	rn.sid = TC_UUIDGenerator::getInstance()->genID();

	int timeout = 10000;

	while(!_terminate)
	{
		time_t now = TNOWMS;

		try
		{
			_adminPrx->tars_set_timeout(timeout/2)->tars_hash(tars::hash<string>()(rn.nodeName))->reportNode(rn);
		}
		catch(exception &ex)
		{
			TLOG_ERROR("report admin, error:" << ex.what() << endl);
		}

		int64_t diff = timeout-(TNOWMS-now);

		if(diff > 0 )
		{
			std::unique_lock<std::mutex> lock(_mutex);

			if(diff > timeout)
			{
				diff = timeout;
			}
			_cond.wait_for(lock, std::chrono::milliseconds(diff));
		}
	}
}

int ServerManager::startServer( const string& application, const string& serverName, string &result)
{
	return ServerObject::startServer(application, serverName, result);
}

int ServerManager::stopServer( const string& application, const string& serverName, string &result)
{
	return ServerObject::stopServer(application, serverName, result);
}

int ServerManager::notifyServer( const string& application, const string& serverName, const string &message, string &result)
{
	return ServerObject::notifyServer(application, serverName, message, result);
}
