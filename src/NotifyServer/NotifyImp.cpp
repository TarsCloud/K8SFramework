#include "NotifyImp.h"
#include "NotifyMsgQueue.h"

static std::string getNotifyLevel(const std::string &sNotifyMessage) {
    std::string Alarm = "[alarm]";
    std::string Error = "[error]";
    std::string Warn = "[warn]";
    std::string Fail = "[fail]";

    if (sNotifyMessage.find(Alarm) != std::string::npos) return Alarm;
    if (sNotifyMessage.find(Error) != std::string::npos) return Error;
    if (sNotifyMessage.find(Warn) != std::string::npos) return Warn;
    if (sNotifyMessage.find(Fail) != std::string::npos) return Error;

    return "[normal]";
}

void NotifyImp::loadConf() {
    // extern TC_Config *g_pconf;
    // TC_DBConf tcDBConf;
    // tcDBConf.loadFromMap(g_pconf->getDomainMap("/tars/db"));
    // _mysqlConfig.init(tcDBConf);
}

void NotifyImp::initialize() {
    loadConf();
}

void NotifyImp::reportServer(const string &sServerName,
                             const string &sThreadId,
                             const string &sResult,
                             tars::CurrentPtr current) {
                                 
    std::string sPodId = current->getHostName();
    LOG->debug() << "reportServer|" << sServerName << "|" << sPodId << "|" << sThreadId << "|" << sResult << endl;
    DLOG << "reportServer|" << sServerName << "|" << sPodId << "|" << sThreadId << "|" << sResult << endl;

    vector<string> v = TC_Common::sepstr<string>(sServerName, ".");

    NotifyRecord notifyRecord;
    notifyRecord.app = v[0];
    notifyRecord.server = v.size() < 2? "" : v[1];
    notifyRecord.podName = sPodId;
    notifyRecord.impThread = sThreadId;
    notifyRecord.level = getNotifyLevel(sResult);
    notifyRecord.message = sResult;
    notifyRecord.notifyTime = TNOW;
    notifyRecord.source = "program";

    NotifyMsgQueue::getInstance()->add(notifyRecord);
}

void NotifyImp::notifyServer(const string &sServerName, tars::NOTIFYLEVEL level, 
                               const string &sMessage, tars::CurrentPtr current) {
    std::string sPodName = current->getHostName();

    // std::string sPodId = current->getContext()["SERVER_HOST_NAME"];

    vector<string> v = TC_Common::sepstr<string>(sServerName, ".");
    
    NotifyRecord notifyRecord;
    notifyRecord.app = v[0];
    notifyRecord.server = v.size() < 2? "" : v[1];
    notifyRecord.podName = sPodName;
    notifyRecord.impThread = "";
    notifyRecord.level = etos(level);
    //cmd;
    notifyRecord.message = sMessage;
    notifyRecord.notifyTime = TNOW;
    notifyRecord.source = "program";

    NotifyMsgQueue::getInstance()->add(notifyRecord);
}

tars::Int32 NotifyImp::getNotifyInfo(const tars::NotifyKey & stKey,tars::NotifyInfo &stInfo,tars::TarsCurrentPtr current)
{
    return 0;
}

void NotifyImp::reportNotifyInfo(const tars::ReportInfo & info, tars::TarsCurrentPtr current)
{
    return;
}
