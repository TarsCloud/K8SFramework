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

#include "QueryServer.h"
#include "QueryImp.h"
#include "ESHelper.h"

void QueryServer::initialize()
{
	const auto& config = getConfig();
	ESHelper::setAddressByTConfig(config);
	std::string indexPre{};
	if (ServerConfig::ServerName == "tarsqueryproperty")
	{
		indexPre = config.get("/tars/elk/indexpre<property>");
	}
	else if (ServerConfig::ServerName == "tarsquerystat")
	{
		indexPre = config.get("/tars/elk/indexpre<stat>");
	}
	else
	{
		throw std::runtime_error("unexpected tars servername");
	}

	if (indexPre.empty())
	{
		TLOGERROR("get empty elk index prefix");
		std::cout << "get empty elk index prefix" << std::endl;
		throw std::runtime_error("get empty elk index prefix");
	}
	QueryImp::setIndexPre(indexPre);

	addServant<QueryImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".QueryObj");
}

void QueryServer::destroyApp()
{
}
