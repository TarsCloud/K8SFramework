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

#include "AdminRegistryServer.h"
#include "AdminRegistryImp.h"

TC_Config * g_pconf;
AdminRegistryServer g_app;

extern TC_Config * g_pconf;

void AdminRegistryServer::initialize()
{
    TLOG_DEBUG("AdminRegistryServer::initialize..." << endl);

    addServant<AdminRegistryImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".AdminRegObj");

    TLOG_DEBUG("AdminRegistryServer::initialize OK!" << endl);

	_ast.setTimeout(10000);
	_ast.start();
}

void AdminRegistryServer::destroyApp()
{
	_ast.terminate();

	TLOG_DEBUG("AdminRegistryServer::destroyApp ok" << endl);
}

int main(int argc, char *argv[])
{
    try
    {
        g_app.main(argc, argv);

        g_app.waitForShutdown();
    }
    catch(exception &ex)
    {
        cerr<< ex.what() << endl;
    }

    return 0;
}


