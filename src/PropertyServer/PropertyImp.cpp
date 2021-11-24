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

#include "PropertyImp.h"
#include "PropertyPushGateway.h"

///////////////////////////////////////////////////////////
void PropertyImp::initialize()
{
}

int PropertyImp::reportPropMsg(const map<StatPropMsgHead, StatPropMsgBody>& propMsg, tars::CurrentPtr current)
{
	TLOGDEBUG("PropertyImp::reportPropMsg size:" << propMsg.size() << ", " << current->getIp() << endl);
	for (auto& item: propMsg)
	{
		const static std::set<std::string> filterModules = { "tars.tarsnode" };
		auto&& moduleName = item.first.moduleName;
		if (filterModules.find(moduleName) != filterModules.end())
		{
			continue;
		}
		auto head = item.first;
		head.ip = current->getIp();
		PropertyPushGateway::instance().push(head, item.second);
	}
	return 0;
}

