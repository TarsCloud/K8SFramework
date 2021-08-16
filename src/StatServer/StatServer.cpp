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

#include "StatServer.h"
#include "StatImp.h"
#include "StatPushGateway.h"

void StatServer::initialize() {
    try {
        //关闭远程日志
        RemoteTimeLogger::getInstance()->enableRemote("", false);

        //增加对象
        addServant<StatImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".StatObj");

        const auto &config = Application::getConfig();

        vector<string> vIpGroup = config.getDomainKey("/tars/masteripgroup");
        for (auto &item : vIpGroup) {
            vector<string> vOneGroup = TC_Common::sepstr<string>(item, ";", true);
            if (vOneGroup.size() < 2) {
                TLOGERROR("StatImp::initialize wrong masteripgroup:" << item << endl);
                continue;
            }
            _mVirtualMasterIp[vOneGroup[0]] = vOneGroup[1];
        }
        StatPushGateway::instance().init(config);
        StatPushGateway::instance().start();
    }
    catch (exception &ex) {
        TLOGERROR("StatServer::initialize catch exception:" << ex.what() << endl);
        exit(0);
    }
    catch (...) {
        TLOGERROR("StatServer::initialize catch unknown exception  " << endl);
        exit(0);
    }
}

map<string, string> &StatServer::getVirtualMasterIp() {
    return _mVirtualMasterIp;
}

void StatServer::destroyApp() {
    TLOGDEBUG("StatServer::destroyApp ok" << endl);
}
