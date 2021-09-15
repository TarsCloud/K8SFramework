
#include "ServerObject.h"
#include <servant/Application.h>
#include <RegistryServer/NodeDescriptor.h>
#include "Fixed.h"
#include "ProxyManger.h"
#include "Launcher.h"
#include "ContainerDetail.h"

int ServerObject::MetaData::updateMetaDataFromDescriptor(const ServerDescriptor &descriptor) {
    return updateConfFromDescriptor(descriptor);
}

int ServerObject::MetaData::updateConfFromDescriptor(const ServerDescriptor &descriptor) {

    _vAdapter.clear();
    map<string, string> m;

    TC_Config tConf;
    try {
        m["node"] = FIXED_NODE_PROXY_NAME;
        tConf.insertDomainParam("/tars/application/server", m, true);

        for (const auto &pair: descriptor.adapters) {
            _vAdapter.push_back(pair.first);
            m.clear();
            m["endpoint"] = TC_Common::replace(pair.second.endpoint, "${localip}", container_detail::listenAddress);
            m["allow"] = pair.second.allowIp;
            m["queuecap"] = TC_Common::tostr(pair.second.queuecap);
            m["queuetimeout"] = TC_Common::tostr(pair.second.queuetimeout);
            m["maxconns"] = TC_Common::tostr(pair.second.maxConnections);
            m["threads"] = TC_Common::tostr(pair.second.threadNum);
            m["servant"] = TC_Common::tostr(pair.second.servant);
            m["protocol"] = pair.second.protocol;
            tConf.insertDomainParam("/tars/application/server/" + pair.first, m, true);
        }
        _vAdapter.emplace_back("AdminAdapter");

        m.clear();
        m["${modulename}"] = _sServerApp + "." + _sServerName;
        m["${app}"] = _sServerApp;
        m["${server}"] = _sServerName;
        m["${basepath}"] = container_detail::imageBindServerBinDir;
        m["${datapath}"] = container_detail::imageBindServerDataDir;
        m["${logpath}"] = container_detail::imageBindServerLogDir;
        m["${localip}"] = container_detail::listenAddress;
        m["${locator}"] = FIXED_QUERY_PROXY_NAME;
        m["${local}"] = "tcp -h 127.0.0.1 -p " + std::to_string(_adminPort) + " -t 10000";
        m["${asyncthread}"] = TC_Common::tostr(descriptor.asyncThreadNum);
        m["${mainclass}"] = "com.qq." + TC_Common::lower(_sServerApp) + "." + TC_Common::lower(_sServerName) + "." + _sServerName;
        m["${enableset}"] = "n";
        m["${setdivision}"] = "NULL";

        string sProfile = descriptor.profile;
        sProfile = TC_Common::replace(sProfile, m);

        TC_Config tProfileConf;
        tProfileConf.parseString(sProfile);
        tConf.joinConfig(tProfileConf, true);

        string sStream = TC_Common::replace(tConf.tostr(), "\\s", " ");

        ofstream configFile(_sServerConfFile.c_str());
        if (!configFile.good()) {
            std::cout << "cannot open configuration file: " + _sServerConfFile;
            return -1;
        }

        configFile << sStream;
        configFile.close();

        return loadConf(tConf);

    } catch (TC_Config_Exception &e) {
        LOG->error() << FILE_FUN << e.what() << endl;
        return -1;
    }
}

int ServerObject::MetaData::loadConf(TC_Config &tConf) {
    constexpr char TARS_JAVA[] = "tars_java";
    try {
        if (strncmp(_sServerType.c_str(), TARS_JAVA, sizeof(TARS_JAVA) - 1) == 0) {
            std::string sJvmParams = tConf.get("/tars/application/server<jvmparams>", "");
            _sServerLauncherArgv = TC_Common::replace(_sServerLauncherArgv, "#{jvmparams}", sJvmParams);

            std::string sMainClass = tConf.get("/tars/application/server<mainclass>", "");
            _sServerLauncherArgv = TC_Common::replace(_sServerLauncherArgv, "#{mainclass}", sMainClass);

            std::string sClassPath = tConf.get("/tars/application/server<classpath>", "");
            _sServerLauncherArgv = TC_Common::replace(_sServerLauncherArgv, "#{classpath}", sClassPath);
        }

        _sServerLauncherEnv = tConf.get("/tars/application/server<env>", "");
        _iTimeout = TC_Common::strto<int>(tConf.get("/tars/application/server<hearttimeout>", "60"));
    }
    catch (const exception &e) {
        LOG->error() << FILE_FUN << e.what() << endl;
        return -1;
    }

    return 0;
}

ServerObject::ProcessStatus ServerObject::_doStart(std::string &result) {
    assert(_runtimeData.presentState == Inactive);
    bool activateOk = false;
    do {
        try {
            _runtimeData.presentState = Activating;
            updateServerState(_runtimeData.settingState, _runtimeData.presentState);

            auto registryProxy = ProxyManger::instance().getRegistryProxy();
            if (!registryProxy) {
                LOG->error() << FILE_FUN << "getRegistryProxy \"" << _metaData._sServerApp << "." << _metaData._sServerName << "\" error " << endl;
                break;
            }

            ServerDescriptor descriptor;

            int res = registryProxy->getServerDescriptor(_metaData._sServerApp, _metaData._sServerName, descriptor);
            LOG->error() << "get getServerDescriptor" << descriptor.writeToJsonString() << std::endl;
            if (res == -1) {
                LOG->error() << FILE_FUN << "getServer \"" << _metaData._sServerApp << "." << _metaData._sServerName << "\" error " << endl;
                break;
            }

            res = _metaData.updateMetaDataFromDescriptor(descriptor);

            if (res == -1) {
                LOG->error() << FILE_FUN << "updateMetaDataFromDescriptor \"" << _metaData._sServerApp << "." << _metaData._sServerName << "\" error " << endl;
                break;
            }

            vector<string> vArgv = TC_Common::sepstr<string>(_metaData._sServerLauncherArgv, " ");
            vector<string> vEnvs;

            //todo set vEnvs;

            pid_t pid = Launcher::activate(_metaData._sServerLauncherFile, _metaData._sServerBaseDir, _metaData._sStdout_Stderr, vArgv, vEnvs);

            if (pid > MIN_PID_VALUE) {
                _runtimeData._pid = pid;
                _runtimeData._vAdapterKeepTimes.clear();
                time_t now = TNOW;
                for (auto &&adapter:_metaData._vAdapter) {
                    _runtimeData._vAdapterKeepTimes.emplace_back(adapter, now);
                }
                _runtimeData._tLastStartTime = TNOW;
                _runtimeData.presentState = Activating;
                activateOk = true;
                break;
            }
        } catch (std::exception &e) {
            LOG->error() << FILE_FUN << "_startServer \"" << _metaData._sServerApp << "." << _metaData._sServerName << "\"" << e.what() << endl;
            break;
        }
        LOG->error() << FILE_FUN << "activate \"" << _metaData._sServerApp << "." << _metaData._sServerName << "\" error " << endl;
    } while (false);

    if (!activateOk) {
        _runtimeData.presentState = tars::ServerState::Inactive;
        updateServerState(_runtimeData.settingState, _runtimeData.presentState);
        result = "activate server error, " + _metaData._sServerApp + ":" + _metaData._sServerName;
        return ProcessStatus::Error;
    }
    return ProcessStatus::Processing;
}

int ServerObject::_startServer(std::string &result) {

    if (_runtimeData.presentState == Active || _runtimeData.presentState == Activating) { ;
        return 0;
    }

    if (_runtimeData.presentState == Deactivating) {
        result = "server is deactivating, can't start now...";
        return -1;
    }

    assert(_runtimeData.presentState == Inactive);

    return _doStart(result);
}

int ServerObject::startServer(std::string &result) {
    lock_guard<std::mutex> lockGuard(_mutex);
    _runtimeData.settingState = Active;
    return _startServer(result);
}

int ServerObject::_stopServer(std::string &result) {

    if (_runtimeData.presentState == Inactive || _runtimeData.presentState == Deactivating) {
        return 0;
    }

    if (_runtimeData.presentState == Activating) {
        result = "server is activating, can't stop now...";
        return -1;
    }

    ProcessStatus status = _doStop(result);
    if (status == Error) {
        return -1;
    }

    if (status == Processing) {
        _waitingStop();
    }
    return 0;
}

int ServerObject::stopServer(std::string &result) {
    lock_guard<std::mutex> lockGuard(_mutex);
    _runtimeData.settingState = Inactive;
    return _stopServer(result);
}


int ServerObject::restartServer(std::string &result) {
    lock_guard<std::mutex> lockGuard(_mutex);
    _runtimeData.settingState = Active;
    return _stopServer(result);
}

int ServerObject::notifyServer(const string &sCommand, string &sResult) {
    string sAdminPrxName = "AdminObj@tcp -h 127.0.0.1 -p " + to_string(_metaData._adminPort) + " -t 30000";
    try {
        AdminFPrx pAdminPrx = ProxyManger::createAdminProxy(sAdminPrxName);
        sResult = pAdminPrx->notify(sCommand);
    } catch (const exception &e) {
        sResult = "error" + string(e.what());
        return -1;
    }
    return 0;
}

//此函数期望被周期性调用
void ServerObject::checkServerState() {

    lock_guard<std::mutex> lockGuard(_mutex);

    if (_runtimeData.presentState == Activating) {
        time_t now = TNOW;
        //启动后的 6 秒内不做检查, 适应启动速度比较慢的情况
        if (_runtimeData._tLastStartTime + 6 >= now) {
            return;
        }
        bool bTimeout = false;
        for (const auto &item:_runtimeData._vAdapterKeepTimes) {
            if (item.second + _metaData._iTimeout < now) {
                LOG->error() << "call server Timeout ," << item.second << "|" << _metaData._iTimeout << "|" << now << endl;
                bTimeout = true;
            }
        }

        if (bTimeout) {
            if (_runtimeData.presentState == Activating) {
                _runtimeData.presentState = Active;
            }
            string result;
            _stopServer(result);
            return;
        }
    }


    if (_runtimeData.settingState == Active) {
        if (_runtimeData.presentState == Deactivating) {
            // 此种情况表示 正在执行 重启 命令.待_runtimeData.presentState变更为 Inactive后再启动.
            return;
        }

        if (_runtimeData.presentState == Activating) {
        }

        time_t now = TNOW;
        //启动后的 6 秒内不做检查, 适应启动速度比较慢的情况
        if (_runtimeData._tLastStartTime + 6 >= now) {
            return;
        }

        bool bStopped = _runtimeData.presentState == Inactive || _checkStopped();
        if (bStopped) {
            if (_isAutoStart()) {
                string result;
                _startServer(result);
            }
            return;
        }

        bool bTimeout = false;
        for (const auto &item:_runtimeData._vAdapterKeepTimes) {
            if (item.second + _metaData._iTimeout < now) {
                LOG->error() << "call server Timeout ," << item.second << "|" << _metaData._iTimeout << "|" << now << endl;
                bTimeout = true;
            }
        }

        if (bTimeout) {
            if (_runtimeData.presentState == Activating) {
                _runtimeData.presentState = Active;
            }
            string result;
            _stopServer(result);
            return;
        }
    }
}

void ServerObject::updateKeepAliveTime(const string &adapter) {
    lock_guard<std::mutex> lockGuard(_mutex);

    if (_runtimeData.presentState == Activating) {
        _runtimeData.presentState = Active;
        updateServerState(_runtimeData.settingState, Active);
    }

    if (adapter.empty()) {
        for (auto &item:_runtimeData._vAdapterKeepTimes) {
            item.second = TNOW;
        }
        return;
    }

    for (auto &item:_runtimeData._vAdapterKeepTimes) {
        if (item.first == adapter) {
            item.second = TNOW;
            return;
        }
    }

    LOG->error() << "Not Match adapter " << adapter << "|" << TNOW << endl;
}

void ServerObject::updateKeepActiving() {
    lock_guard<std::mutex> lockGuard(_mutex);

    if (_runtimeData.presentState != Activating) {
        _runtimeData.presentState = Activating;
        updateServerState(_runtimeData.settingState, Activating);
    }

    for (auto &item:_runtimeData._vAdapterKeepTimes) {
        item.second = TNOW;
    }
};

bool ServerObject::checkStopped() {
    lock_guard<std::mutex> lockGuard(_mutex);
    return _checkStopped();
}

int ServerObject::addFile(string sFile, std::string &result) {
    lock_guard<std::mutex> lockGuard(_mutex);
    assert(!sFile.empty());

    if (!TC_File::isAbsolute(sFile)) {
        sFile = _metaData._sServerBaseDir + "bin" + FILE_SEP + sFile;
    }

    sFile = TC_File::simplifyDirectory(TC_Common::trim(sFile));

    string sFilePath = TC_File::extractFilePath(sFile);

    string sFileName = TC_File::extractFileName(sFile);

    TarsRemoteConfig tTarsRemoteConfig;
    tTarsRemoteConfig.setConfigInfo(Application::getCommunicator(), ServerConfig::Config, _metaData._sServerApp, _metaData._sServerName, _metaData._sServerBaseDir, "");
    return tTarsRemoteConfig.addConfig(sFileName, result);
}

void ServerObject::_kill() {
    if (_runtimeData._pid >= MIN_PID_VALUE) {
        ::kill(_runtimeData._pid, SIGKILL);
        usleep(2 * 1000); // 2ms
        int stat;
        ::waitpid(-1, &stat, WNOHANG);
    }
    _runtimeData._pid = -1;
    _runtimeData.presentState = Inactive;
}

void ServerObject::kill() {
    lock_guard<std::mutex> lockGuard(_mutex);
    _kill();
}

void ServerObject::uploadStopStat(int stopStat) {

    std::string appServer = _metaData._sServerApp + "." + _metaData._sServerName;
    std::string message;

    if (WIFEXITED(stopStat)) {
        int code = WEXITSTATUS(stopStat);  // 主动调用 return 或 exit 退出
        message = string("[alarm] server exit with code ") + to_string(code);
    } else if (WIFSIGNALED(stopStat)) {
        int signal = WTERMSIG(stopStat);   // 接收到信号退出
        message = string("[alarm] server exit with signal ") + to_string(signal);
    }

    if (message.empty()) {
        return;
    }

    try {
        auto notifyFPrx = ProxyManger::instance().getNotifyProxy();
        if (!notifyFPrx) {
            LOG->error() << "get NotifyPrx error" << endl;
            return;
        }
        std::map<string, string> context;
        context.insert(make_pair("SERVER_HOST_NAME", container_detail::podName));
        notifyFPrx->reportServer(appServer, "-1", message, context);
    } catch (const exception &e) {
        LOG->error() << "call notifyFPrx->reportServer() catch exception :" << e.what() << endl;
        return;
    }
}

bool ServerObject::_checkStopped() {
    if (_runtimeData._pid < MIN_PID_VALUE) {
        _runtimeData.presentState = Inactive;
        return true;
    }

    int iStat;
    pid_t ret = waitpid(_runtimeData._pid, &iStat, WNOHANG);
    if (ret > MIN_PID_VALUE) {
        LOG->debug() << "wait pid return  " << ret << " " << _runtimeData._pid << endl;
        uploadStopStat(iStat);
        _runtimeData._pid = -1;
        _runtimeData.presentState = Inactive;
        return true;
    }

    if (ret < 0) {
        LOG->debug() << "wait pid return " << ret << " " << _runtimeData._pid << endl;
        _runtimeData._pid = -1;
        _runtimeData.presentState = Inactive;
        return true;
    }

    return false;
}

bool ServerObject::_isAutoStart() {
    time_t now = TNOW;
    assert(now >= _runtimeData._tLastStartTime);
    return (_runtimeData._tLastStartTime + START_SERVER_INTERVAL_TIME <= now);
}

void ServerObject::updateServerState() {
    try {
        auto ptr = ProxyManger::instance().getRegistryProxy();
        if (!ptr) {
            LOG->debug() << "get RegistryProxy error" << endl;
            return;
        }
        std::string appServer = _metaData._sServerApp + "-" + _metaData._sServerName;
        ServerStateInfo info;
        info.settingState = _runtimeData.settingState;
        info.presentState = _runtimeData.presentState;
        LOG->debug() << "Update State " << info.settingState << "|" << info.presentState << endl;
        ptr->updateServerState(container_detail::podName, etos(_runtimeData.settingState), etos(_runtimeData.presentState));
    } catch (const exception &e) {
        LOG->debug() << "updateServerState exception : " << e.what() << endl;
    }
}

void ServerObject::updateServerState(ServerState settingState, tars::ServerState presentState) {
    TimerTaskQueue::instance().pushTimerTask([settingState, presentState] {
        try {
            auto ptr = ProxyManger::instance().getRegistryProxy();
            if (!ptr) {
                LOG->debug() << "get RegistryProxy error" << endl;
                return;
            }
            std::string appServer = container_detail::imageBindServerApp + "-" + container_detail::imageBindServerName;
            ServerStateInfo info;
            info.settingState = settingState;
            info.presentState = presentState;
            LOG->debug() << "Update State " << info.settingState << "|" << info.presentState << endl;
            ptr->updateServerState(container_detail::podName, etos(settingState), etos(presentState));
        } catch (const exception &e) {
            LOG->debug() << "updateServerState exception : " << e.what() << endl;
        }
    }, 0);
}


ServerObject::ProcessStatus ServerObject::_doStop(std::string &result) {
    bool bStopped = _checkStopped();
    if (bStopped) {
        _runtimeData.presentState = Inactive;
        updateServerState(_runtimeData.settingState, _runtimeData.presentState);
        return ServerObject::Done;
    }

    _runtimeData.presentState = Deactivating;
    updateServerState(_runtimeData.settingState, _runtimeData.presentState);

    do {
        if (_metaData._sServerType != SERVERTYPE_NOT_TARS) {
            string sAdminPrx = "AdminObj@tcp -h 127.0.0.1 -p " + to_string(_metaData._adminPort) + " -t 3000";
            try {
                AdminFPrx pAdminPrx = ProxyManger::createAdminProxy(sAdminPrx);
                LOG->debug() << FILE_FUN << _metaData._sServerApp << "." << _metaData._sServerName << " call " << sAdminPrx << endl;
                pAdminPrx->async_shutdown(nullptr);
                break;
            } catch (...) {
                LOG->debug() << FILE_FUN << _metaData._sServerApp << "." << _metaData._sServerName << " by async_shutdown fail:|" << "|will use kill -9" << endl;
            }
        }
        //todo 先用 SIGTERM ,等待若干秒后，再看是否需要  用  SIGKILL
        _kill();
        _runtimeData.presentState = Inactive;
        updateServerState(_runtimeData.settingState, _runtimeData.presentState);
        return ServerObject::Done;
    } while (false);

    return ServerObject::Processing;
}

void ServerObject::_waitingStop() {

    assert(_runtimeData.presentState == Deactivating);

    auto selfPtr = shared_from_this();
    TimerTaskQueue::instance().pushCycleTask(
            [selfPtr](const size_t &callTimes, size_t &callCycle) {
                if (selfPtr->checkStopped()) {
                    selfPtr->updateServerState();
                    callCycle = 0;
                    return;
                }
                if (callTimes * callCycle >= MAX_DEACTIVATING_TIME) {
                    selfPtr->kill();
                    selfPtr->updateServerState();
                    callCycle = 0;
                    return;
                }
            }, STOPPED_CHECK_INTERVAL, STOPPED_CHECK_INTERVAL);
}



