//
// Created by jarod on 2022/5/7.
//

#ifndef FRAMEWORK_SERVERMANAGER_H
#define FRAMEWORK_SERVERMANAGER_H

#include "AdminReg.h"
#include "util/tc_thread.h"
#include "util/tc_singleton.h"

using namespace tars;

class ServerManager : public TC_Thread ,public TC_Singleton<ServerManager>
{
public:
	/**
	 *
	 * @param adminObj
	 */
	void initialize(const string &adminObj);

	/**
	 *
	 */
	void terminate();

	/**
	 * 启动指定服务
	 * @param application    服务所属应用名
	 * @param serverName  服务名
	 * @return  int
	 */
	int startServer( const string& application, const string& serverName, string &result) ;

	/**
	 * 停止指定服务
	 * @param application    服务所属应用名
	 * @param serverName  服务名
	 * @return  int
	 */
	int stopServer( const string& application, const string& serverName, string &result) ;

	/**
	 * 通知服务
	 * @param application
	 * @param serverName
	 * @param result
	 * @param current
	 *
	 * @return int
	 */
	int notifyServer( const string& application, const string& serverName, const string &command, string &result);

protected:
	virtual void run();

	void createAdminPrx();
protected:
	bool _terminate = false;
	std::mutex _mutex;
	std::condition_variable _cond;
	string _adminObj;
	AdminRegPrx _adminPrx;
//	map<string, AdminRegPrx> _adminPrxs;
};


#endif //FRAMEWORK_SERVERMANAGER_H
