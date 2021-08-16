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

using namespace std;

void QueryServer::initialize() {
    try {
        const auto &config = Application::getConfig();
        elkIndexPre = config.get("/tars/elk<indexPre>");
        vector<string> elkNodes = config.getDomainKey("/tars/elk/nodes");
        if (elkNodes.empty()) {
            TLOGERROR("QueryServer::initialize empty elk nodes " << endl);
            exit(0);
        }

        for (auto &item : elkNodes) {
            vector<string> vOneNode = TC_Common::sepstr<string>(item, ":", true);
            if (vOneNode.size() < 2) {
                TLOGERROR("QueryServer::initialize wrong elk nodes:" << item << endl);
                exit(0);
            }
            auto port = std::stoi(vOneNode[1]);
            if (port <= 0 || port >= 65535) {
                TLOGERROR("QueryServer::initialize wrong elk nodes:" << item << endl);
                exit(0);
            }

            TLOG_DEBUG("node:" << vOneNode[0] << ":" << port << endl);
            elkTupleNodes.emplace_back(vOneNode[0], port);
        }
        addServant<QueryImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".QueryObj");

        std::srand(std::time(nullptr));
    }
    catch (exception &ex) {
        TLOGERROR("QueryServer::initialize catch exception:" << ex.what() << endl);
        exit(0);
    }
    catch (...) {
        TLOGERROR("QueryServer::initialize catch unknown exception  " << endl);
        exit(0);
    }
}
/////////////////////////////////////////////////////////////////

/////////////////////////////////////////////////////////////////
void QueryServer::destroyApp() {
}

const std::string &QueryServer::getELKIndexPre() const {
    return elkIndexPre;
}
/////////////////////////////////////////////////////////////////

void QueryServer::getELKNodeAddress(string &host, int &port) {
    std::lock_guard<std::mutex> lockGuard(mutex);
    if (elkTupleNodes.empty()) {
        throw std::runtime_error(
                std::string("fatal error: empty elk node addresses"));
    }
    auto tuple = elkTupleNodes[std::rand() % elkTupleNodes.size()];
    host = std::get<0>(tuple);
    port = std::get<1>(tuple);
}

QueryServer g_app;

int
main(int argc, char *argv[]) {
    try {
        g_app.main(argc, argv);
        RemoteTimeLogger::getInstance()->enableRemote("inout", false);
        g_app.waitForShutdown();
    }
    catch (std::exception &e) {
        cerr << "std::exception:" << e.what() << std::endl;
    }
    catch (...) {
        cerr << "unknown exception." << std::endl;
    }
    return -1;
}
/////////////////////////////////////////////////////////////////
