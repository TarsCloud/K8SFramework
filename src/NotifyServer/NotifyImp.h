#ifndef NOTIFY_I_H
#define NOTIFY_I_H

#include "servant/NotifyF.h"
#include "util/tc_common.h"
#include "util/tc_config.h"
// #include "util/tc_mysql.h"
#include "servant/RemoteLogger.h"

using namespace tars;

class NotifyImp : public Notify {
public:
    /**
     * 初始化
     *
     * @return int
     */
    void initialize() final;

    /**
     * 退出
     */
    void destroy() final {};

    /**
     * report
     * @param sServerName
     * @param sThreadId
     * @param sResult
     * @param current
     */
    void reportServer(const string &sServerName,
                      const string &sThreadId,
                      const string &sResult,
                      tars::TarsCurrentPtr current) override;

    /**
     * notify
     * @param sServerName
     * @param sThreadId
     * @param sCommand
     * @param sResult
     * @param current
     */
    virtual void notifyServer(const string& sServerName, NOTIFYLEVEL level, const string& sMessage, tars::TarsCurrentPtr current);

    /**
     * get notify info
     */
    virtual tars::Int32 getNotifyInfo(const tars::NotifyKey & stKey,tars::NotifyInfo &stInfo,tars::TarsCurrentPtr current);

    /*
     *reportNotifyInfo
     *@param info
     */
    virtual void reportNotifyInfo(const tars::ReportInfo & info, tars::TarsCurrentPtr current);

    // void notifyServerEx(const std::string &sServerName, tars::NOTIFYLEVEL level, const std::string &sTitle,
    //                     const std::string &sMessage, tars::CurrentPtr current) override;

protected:
    void loadConf();

protected:
    // TC_Mysql _mysqlConfig;
};

#endif
