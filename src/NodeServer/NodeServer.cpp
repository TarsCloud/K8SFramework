

#include "NodeServer.h"
#include "NodeImp.h"
#include "ServerImp.h"
#include "TimerTaskQueue.h"
#include "ServerObject.h"
#include "ProxyManger.h"
#include "Fixed.h"
#include "ContainerDetail.h"
#include "ServerManger.h"

void NodeServer::initialize() {
    if (!container_detail::loadContainerDetailFromEnv()) {
        LOG->error() << "loadContainerDetailFromEnv error";
        cerr << "loadContainerDetailFromEnv error";
        exit(-1);
    }

    try {
        addServant<NodeImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".NodeObj");
        addServant<ServerImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".ServerObj");
        TC_Config &gConf = NodeServer::getConfig();

        string sNotifyProxyName = gConf.get("/tars/application/server<notify>", FIXED_NOTIFY_PROXY_NAME);
        ProxyManger::instance().setNotifyObjName(sNotifyProxyName);

        string sQueryProxyName = gConf.get("/tars/application/client<locator>", FIXED_QUERY_PROXY_NAME);
        ProxyManger::instance().setQueryObjName(sQueryProxyName);

        string sRegistryProxyName = gConf.get("/tars/node<registryObj>", FIXED_REGISTRY_PROXY_NAME);
        ProxyManger::instance().setRegistryObjName(sRegistryProxyName);

    } catch (TC_Exception &ex) {
        LOG->error() << "NodeServer initialize exception: " << ex.what() << endl;
        cerr << "NodeServer initialize exception: " << ex.what() << endl;
        exit(-1);
    }

    _timerTaskThread = std::thread(
            []() {
                TimerTaskQueue::instance().run();
                cerr << "TimerTaskQueue thread stopped , Program Will stop" << endl;
                exit(-1);
            });
    _timerTaskThread.detach();

    const std::string &serverApp = container_detail::imageBindServerApp;
    const std::string &serverName = container_detail::imageBindServerName;

    try {
        auto pRegistryObj = ProxyManger::instance().getRegistryProxy();
        pRegistryObj->updateServerState(container_detail::podName, etos(Active), etos(Activating));

        auto pServerObj = std::make_shared<ServerObject>();
        if (pServerObj == nullptr) {
            LOG->error() << "call std::make_shared<ServerObject><" << serverApp << "," << serverName << "> error";
            cerr << "call std::make_shared<ServerObject><" << serverApp << "," << serverName << "> error";
            exit(-1);
        }

        assert(pServerObj != nullptr);

        TimerTaskQueue::instance().pushCycleTask(
                [pServerObj](const size_t &, size_t &) {
                    assert(pServerObj != nullptr);
                    pServerObj->updateServerState(); //每间隔一定周期,上报服务状态
                }, 10, 200);

        TimerTaskQueue::instance().pushCycleTask(
                [pServerObj](const size_t &, size_t &) {
                    assert(pServerObj != nullptr);
                    pServerObj->checkServerState();  //每间隔一定周期,检测服务状态.拉起服务
                }, 1, 2);

        ServerManger::instance().putServer(pServerObj);

    } catch (std::exception &ex) {
        LOG->error() << "call NodeServer::initialize() Get Exception : " << ex.what() << endl;
        cerr << "call NodeServer::initialize() Get Exception : " << ex.what() << endl;
        exit(-1);
    }

//    LOG_ERROR << "NodeServer::init ok" << endl;
}

void NodeServer::destroyApp() {
    cout << "NodeServer::destroyApp ok" << endl;
}
