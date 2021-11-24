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

#include "StatImp.h"
#include "StatPushGateway.h"

static std::string fixSlaveName(const string& sSlaveName)
{
//	string::size_type pos = sSlaveName.find('.');
//	if (pos != string::npos)
//	{
//		pos = sSlaveName.find('.', pos + 1);
//		if (pos != string::npos)
//		{
//			return sSlaveName.substr(0, pos);
//		}
//	}
	return sSlaveName;
}

////////////////////////////////////////////////////////
void StatImp::initialize()
{
}

///////////////////////////////////////////////////////////

int StatImp::reportMicMsg(const map<tars::StatMicMsgHead, tars::StatMicMsgBody>& statmsg, bool bFromClient, tars::CurrentPtr current)
{
	TLOGINFO("report---------------------------------access size:" << statmsg.size() << "|bFromClient:" << bFromClient << endl);
	for (auto&& item: statmsg)
	{
		const static std::set<std::string> filterSlaves = { "tars.tarsnode", "tars.tarsproperty", "tars.tarsstat", "tars.tarsnotify" };

		auto&& slaveName = item.first.slaveName;
		if (filterSlaves.find(slaveName) != filterSlaves.end())
		{
			continue;
		}

		auto&& body = item.second;
		//三个数据都为0时不入库
		if (body.count == 0 && body.execCount == 0 && body.timeoutCount == 0)
		{
			continue;
		}

		auto head = item.first;
		string sMasterName = head.masterName;
		string::size_type pos = sMasterName.find('@');
		if (pos != string::npos)
		{
			head.masterName = sMasterName.substr(0, pos);
			const static std::set<std::string> filterMasters = { "tars.tarsnode", "es" };
			if (filterMasters.find(head.masterName) != filterMasters.end())
			{
				continue;
			}
			head.tarsVersion = sMasterName.substr(pos + 1);
		}

		if (bFromClient)
		{
			head.masterIp = current->getIp();  //以前是自己获取主调ip,现在从proxy直接
			head.slaveName = fixSlaveName(head.slaveName);
		}
		else
		{
			head.slaveIp = current->getIp();//现在从proxy直接
		}
		StatPushGateway::instance().push(head, body);
	}
	return 0;
}

int StatImp::reportSampleMsg(const vector<StatSampleMsg>& msg, tars::CurrentPtr current)
{
	return 0;
}