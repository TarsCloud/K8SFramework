#ifndef NOTIFY_I_H
#define NOTIFY_I_H

#include "servant/NotifyF.h"

using namespace tars;

class NotifyImp : public Notify
{
public:
	/**
	 * 初始化
	 *
	 * @return int
	 */
	void initialize() override
	{
	};

	/**
	 * 退出
	 */
	void destroy() override
	{
	};

	/**
	 * report
	 * @param sServerName
	 * @param sThreadId
	 * @param sResult
	 * @param current
	 */
	void reportServer(const string& sServerName, const string& sThreadId, const string& sResult, tars::TarsCurrentPtr current) override;

	/**
	 * notify
	 * @param sServerName
	 * @param sThreadId
	 * @param sCommand
	 * @param sResult
	 * @param current
	 */
	void notifyServer(const string& sServerName, NOTIFYLEVEL level, const string& sMessage, tars::TarsCurrentPtr current) override;

	/**
	 * get notify info
	 */
	int getNotifyInfo(const tars::NotifyKey& stKey, tars::NotifyInfo& stInfo, tars::TarsCurrentPtr current) override
	{
		return 0;
	}

	/*
	 *reportNotifyInfo
	 *@param info
	 */
	void reportNotifyInfo(const tars::ReportInfo& info, tars::TarsCurrentPtr current) override;
};

#endif
