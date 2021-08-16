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

#include "PropertyServer.h"
#include "PropertyImp.h"
#include "servant/AppCache.h"
#include "PropertyPushGateway.h"

void PropertyServer::initialize() {

    try {
        //关闭远程日志
        RemoteTimeLogger::getInstance()->enableRemote("", false);
        RemoteTimeLogger::getInstance()->enableRemote("PropertyPool", true);
        //增加对象
        addServant<PropertyImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".PropertyObj");

        const auto &config = Application::getConfig();
        PropertyPushGateway::instance().init(config);
        PropertyPushGateway::instance().start();
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

void PropertyServer::destroyApp() {
    TLOGDEBUG("PropertyServer::destroyApp ok" << endl);
}
