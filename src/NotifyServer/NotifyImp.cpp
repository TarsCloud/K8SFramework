﻿#include "NotifyImp.h"
#include "NotifyMsgQueue.h"
#include "servant/RemoteLogger.h"
#include "Storage.h"

static std::string getDomainName(const std::string& ip)
{
	std::string sDomainName{};
	if (!ip.empty())
	{
		Storage::getPodIPMap([ip, &sDomainName](const std::unordered_map <std::string, std::string>& map_)mutable
		{
			auto iterator = map_.find(ip);
			if (iterator != map_.end())
			{
				sDomainName = iterator->second;
			}
		});
	}
	return sDomainName.empty() ? ip : sDomainName;
}

std::string Alarm = "[alarm]";
std::string Error = "[error]";
std::string Warn = "[warn]";
std::string Fail = "[fail]";
std::string Normal = "[normal]";

static std::string getNotifyLevel(const std::string& sNotifyMessage)
{

	if (sNotifyMessage.find(Alarm) != std::string::npos) return Alarm;
	if (sNotifyMessage.find(Error) != std::string::npos) return Error;
	if (sNotifyMessage.find(Warn) != std::string::npos) return Warn;
	if (sNotifyMessage.find(Fail) != std::string::npos) return Error;

	return Normal;
}
static std::string getNotifyLevel(int level)
{
    switch(level)
    {
        case NOTIFYERROR:
            return Error;
        case NOTIFYWARN:
            return Warn;
        default:
        case NOTIFYNORMAL:
            return Normal;
        }

    return Normal;
}

void NotifyImp::reportServer(const string& sServerName, const string& sThreadId, const string& sResult, tars::TarsCurrentPtr current)
{
	std::string sPodIP = current->getIp();
	LOG->debug() << "reportServer|" << sServerName << "|" << sPodIP << "|" << sThreadId << "|" << sResult << endl;
	DLOG << "reportServer|" << sServerName << "|" << sPodIP << "|" << sThreadId << "|" << sResult << endl;

	vector <string> v = TC_Common::sepstr<string>(sServerName, ".");
	NotifyRecord notifyRecord;
	notifyRecord.app = v[0];
	notifyRecord.server = v.size() < 2 ? "" : v[1];
	notifyRecord.podName = getDomainName(sPodIP);
	notifyRecord.impThread = sThreadId;
	notifyRecord.level = getNotifyLevel(sResult);
	notifyRecord.message = sResult;
	notifyRecord.notifyTime = TC_Common::tm2str(TNOW, "%FT%T%z");
	notifyRecord.source = "program";
	NotifyMsgQueue::getInstance()->add(notifyRecord);
}

void NotifyImp::notifyServer(const string& sServerName, NOTIFYLEVEL level, const string& sMessage, tars::TarsCurrentPtr current)
{
	std::string sPodIP = current->getIp();
	vector <string> v = TC_Common::sepstr<string>(sServerName, ".");
	NotifyRecord notifyRecord;
	notifyRecord.app = v[0];
	notifyRecord.server = v.size() < 2 ? "" : v[1];
	notifyRecord.podName = getDomainName(sPodIP);
	notifyRecord.impThread = "";
	notifyRecord.level = getNotifyLevel(level);
	notifyRecord.message = sMessage;
	notifyRecord.notifyTime = TC_Common::tm2str(TNOW, "%FT%T%z");
	notifyRecord.source = "program";
	NotifyMsgQueue::getInstance()->add(notifyRecord);
}

void NotifyImp::reportNotifyInfo(const tars::ReportInfo& info, tars::TarsCurrentPtr current)
{
	LOG->debug() << "reportNotifyInfo|" << info.sApp << "|" << info.sServer << "|" << info.sNodeName << "|" << info.sMessage << endl;

    string sNodeName = info.sNodeName;
    if (info.sNodeName.empty())
    {
        sNodeName = current->getIp();
    }

	NotifyRecord notifyRecord;
	notifyRecord.app = info.sApp;
	notifyRecord.server = info.sServer;
	notifyRecord.podName = getDomainName(sNodeName);
	notifyRecord.impThread = info.sThreadId;
	notifyRecord.level = getNotifyLevel(info.eLevel);
	notifyRecord.message = info.sMessage;
	notifyRecord.notifyTime = TC_Common::tm2str(TNOW, "%FT%T%z");
	notifyRecord.source = "server";

    TLOG_DEBUG(notifyRecord.writeToJsonString() << endl);
  
    NotifyMsgQueue::getInstance()->add(notifyRecord);
}
