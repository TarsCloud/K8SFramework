#include "NotifyImp.h"
#include "NotifyMsgQueue.h"
#include "servant/RemoteLogger.h"

static std::string getNotifyLevel(const std::string& sNotifyMessage)
{
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

void NotifyImp::reportServer(const string& sServerName, const string& sThreadId, const string& sResult, tars::TarsCurrentPtr current)
{

	std::string sPodId = current->getHostName();
	LOG->debug() << "reportServer|" << sServerName << "|" << sPodId << "|" << sThreadId << "|" << sResult << endl;
	DLOG << "reportServer|" << sServerName << "|" << sPodId << "|" << sThreadId << "|" << sResult << endl;

	vector<string> v = TC_Common::sepstr<string>(sServerName, ".");
	NotifyRecord notifyRecord;
	notifyRecord.app = v[0];
	notifyRecord.server = v.size() < 2 ? "" : v[1];
	notifyRecord.podName = sPodId;
	notifyRecord.impThread = sThreadId;
	notifyRecord.level = getNotifyLevel(sResult);
	notifyRecord.message = sResult;
	notifyRecord.notifyTime = TNOW;
	notifyRecord.source = "program";
	NotifyMsgQueue::getInstance()->add(notifyRecord);
}

void NotifyImp::notifyServer(const string& sServerName, NOTIFYLEVEL level, const string& sMessage, tars::TarsCurrentPtr current)
{
	std::string sPodName = current->getHostName();
	vector<string> v = TC_Common::sepstr<string>(sServerName, ".");
	NotifyRecord notifyRecord;
	notifyRecord.app = v[0];
	notifyRecord.server = v.size() < 2 ? "" : v[1];
	notifyRecord.podName = sPodName;
	notifyRecord.impThread = "";
	notifyRecord.level = etos(level);
	notifyRecord.message = sMessage;
	notifyRecord.notifyTime = TNOW;
	notifyRecord.source = "program";
	NotifyMsgQueue::getInstance()->add(notifyRecord);
}

void NotifyImp::reportNotifyInfo(const tars::ReportInfo& info, tars::TarsCurrentPtr current)
{
	std::string sPodName = current->getHostName();

	LOG->debug() << "reportNotifyInfo|" << info.sApp << "|" << info.sServer << "|" << info.sNodeName << "|" << info.sMessage << endl;

	NotifyRecord notifyRecord;
	notifyRecord.app = info.sApp;
	notifyRecord.server = info.sServer;
	notifyRecord.podName = sPodName;
	notifyRecord.impThread = info.sThreadId;
	notifyRecord.level = etos(info.eLevel);
	notifyRecord.message = info.sMessage;
	notifyRecord.notifyTime = TNOW;
	notifyRecord.source = "server";
	NotifyMsgQueue::getInstance()->add(notifyRecord);
}