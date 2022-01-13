#include "ServerObject.h"
#include "Container.h"
#include "ProxyManger.h"
#include "Fixed.h"
#include "Launcher.h"
#include "RegistryServer/NodeDescriptor.h"
#include "RegistryServer/RegistryDescriptor.h"
#include "servant/Application.h"
#include "servant/RemoteLogger.h"
#include "util/tc_config.h"
#include <mutex>
#include <sys/wait.h>

static void notifyMessage(const std::string& message)
{
    std::string appServer = container::serverApp + "." + container::serverName;
    try
    {
        auto notifyFPrx = ProxyManger::instance().getNotifyProxy();
        if (!notifyFPrx)
        {
            TLOGERROR("get null NotifyPrx" << std::endl;);
            return;
        }
        notifyFPrx->reportServer(appServer, "-1", message);
    }
    catch (const exception& e)
    {
        TLOGERROR("call notifyFPrx->reportServer() catch exception :" << e.what() << std::endl);
        return;
    }
}

static void uploadStopStat(int stopStat)
{
    std::string message{};
    if (WIFEXITED(stopStat))
    {
        int code = WEXITSTATUS(stopStat);
        message = string("[alarm] server exit with code ") + to_string(code);
    }
    else if (WIFSIGNALED(stopStat))
    {
        int signal = WTERMSIG(stopStat);
        message = string("[alarm] server exit with signal ") + to_string(signal);
    }
    if (!message.empty())
    {
        notifyMessage(message);
    }
}

static void updateServerState(tars::ServerState settingState, ServerState presentState, pid_t pid)
{
    auto registryPrx = ProxyManger::instance().getRegistryProxy();
    if (!registryPrx)
    {
        auto message = std::string("get null RegistryProxy");
        notifyMessage("[alarm] " + message);
        TLOGERROR(message << std::endl);
    }
    try
    {
        registryPrx->updateServerState(container::podName, etos(settingState), etos(presentState),
                {{ "PID", std::to_string(pid) }});
    }
    catch (const std::exception& e)
    {
        auto message = std::string(
                "call registryProxy->updateServerState() catch exception: ").append(e.what());
        notifyMessage("[alarm]" + message);
        TLOGERROR(message << std::endl);
    }
}

static bool processExist(pid_t pid)
{
    if (pid < MIN_PID_VALUE)
    {
        return false;
    }
    int stat{};
    auto waitPidRes = ::waitpid(pid, &stat, WNOHANG);
    if (waitPidRes == pid)
    {
        uploadStopStat(stat);
        return false;
    }
    if (waitPidRes < 0)
    {
        if (errno == ECHILD)
        {
            return false;
        }
        TLOGERROR("call ::waitpid get error: " << errno << strerror(errno) << std::endl);
        return true;
    }
    return true;
}

//we will use addition calculation on the VERY_BIG_TIME_VALUE variable,
//so it cannot be used INT32_MAX
constexpr time_t VERY_BIG_TIME_VALUE = { INT32_MAX - 10000 };

enum class ServerTarget
{
    Start,
    Stop,
    Restart
};

struct ServerMetadata
{
public:
    ServerMetadata() :
            serverApp_(container::serverApp),
            serverName_(container::serverName),
            serverType_(container::serverType),
            serverBaseDir_(container::serverBinDir),
            serverConfFile_(container::serverConfFile),
            serverLauncherFile_(container::serverLauncherFile),
            redirect_(container::serverLogDir + "/" + serverApp_ + "/" + serverName_ + "/" +
                      "stdout_stderr"),
            serverLauncherArgv_(container::serverLauncherArgv)
    {
    }

    const std::string& serverApp_;
    const std::string& serverName_;
    const std::string& serverType_;
    const std::string& serverBaseDir_;
    const std::string& serverConfFile_;
    const std::string& serverLauncherFile_;
    const std::string redirect_;

    string serverLauncherArgv_;
    string serverLauncherEnv_;       //环境变量字符串
    int timeout_{ 60 };                 //心跳超时时间
    vector<string> adapters{};       //adapter

    int updateFromDescriptor(const ServerDescriptor& descriptor)
    {
        adapters.clear();
        map<string, string> m{};

        TC_Config tConfig{};
        try
        {
            m["node"] = FIXED_NODE_PROXY_NAME;
            tConfig.insertDomainParam("/tars/application/server", m, true);

            for (const auto& item: descriptor.adapters)
            {
                const auto& adapterName = item.first;
                const auto& adapterDesc = item.second;
                adapters.push_back(adapterName);
                m.clear();
                m["endpoint"] = TC_Common::replace(adapterDesc.endpoint, "${localip}",
                        container::listenAddress);
                m["allow"] = adapterDesc.allowIp;
                m["queuecap"] = TC_Common::tostr(adapterDesc.queuecap);
                m["queuetimeout"] = TC_Common::tostr(adapterDesc.queuetimeout);
                m["maxconns"] = TC_Common::tostr(adapterDesc.maxConnections);
                m["threads"] = TC_Common::tostr(adapterDesc.threadNum);
                m["servant"] = TC_Common::tostr(adapterDesc.servant);
                m["protocol"] = adapterDesc.protocol;
                tConfig.insertDomainParam("/tars/application/server/" + item.first, m, true);
            }
            adapters.emplace_back("AdminAdapter");

            m.clear();
            m["${modulename}"] = serverApp_ + "." + serverName_;
            m["${app}"] = serverApp_;
            m["${server}"] = serverName_;
            m["${basepath}"] = container::serverBinDir + "/";
            m["${datapath}"] = container::serverDataDir + "/";
            m["${logpath}"] = container::serverLogDir + "/";
            m["${localip}"] = container::listenAddress;
            m["${locator}"] = FIXED_QUERY_PROXY_NAME;
            if (container::serverLauncherType == SERVER_BACKGROUND_LAUNCH)
            {
                m["${local}"] = FIXED_LOCAL_ENDPOINT;
            }
            else if (container::serverLauncherType == SERVER_FOREGROUND_LAUNCH)
            {
                m["${local}"] = "tcp -h " + container::listenAddress + " -t 6000 -p " +
                                SERVER_FOREGROUND_ADMIN_PORT;
            }
            else
            {
                assert(false);
            }
            m["${asyncthread}"] = TC_Common::tostr(descriptor.asyncThreadNum);
            m["${mainclass}"] =
                    "com.qq." + TC_Common::lower(serverApp_) + "." + TC_Common::lower(serverName_) +
                    "." + serverName_;
            m["${enableset}"] = "n";
            m["${setdivision}"] = "NULL";

            string sProfile = descriptor.profile;
            sProfile = TC_Common::replace(sProfile, m);

            TC_Config profileConfig;
            profileConfig.parseString(sProfile);
            tConfig.joinConfig(profileConfig, true);

            string configContent = TC_Common::replace(tConfig.tostr(), "\\s", " ");

            ofstream configFile(serverConfFile_.c_str());
            if (!configFile.good())
            {
                std::string message = "cannot open or write configuration file: " + serverConfFile_;
                notifyMessage("[alarm] " + message);
                TLOGERROR(message);
                std::cout << message << std::endl;
                return -1;
            }

            configFile << configContent;
            configFile.close();

            return loadConf(tConfig);
        }
        catch (const std::exception& e)
        {
            std::string message = std::string("parser profile catch exception: ").append(e.what());
            notifyMessage("[alarm] " + message);
            TLOGERROR(message);
            std::cout << message << std::endl;
            return -1;
        }
    }

private:

    int loadConf(const TC_Config& config)
    {
        constexpr char JAVA_TYPE_PREFIX[] = "java-";
        constexpr size_t JAVA_TYPE_PREFIX_SIZE = sizeof(JAVA_TYPE_PREFIX) - 1;
        try
        {
            if (serverType_.compare(0, JAVA_TYPE_PREFIX_SIZE, JAVA_TYPE_PREFIX) == 0)
            {
                std::string jvmParams = config.get("/tars/application/server<jvmparams>", "");
                serverLauncherArgv_ = TC_Common::replace(serverLauncherArgv_, "#{jvmparams}",
                        jvmParams);

                std::string mainClass = config.get("/tars/application/server<mainclass>", "");
                serverLauncherArgv_ = TC_Common::replace(serverLauncherArgv_, "#{mainclass}",
                        mainClass);

                std::string classPath = config.get("/tars/application/server<classpath>", "");
                serverLauncherArgv_ = TC_Common::replace(serverLauncherArgv_, "#{classpath}",
                        classPath);
            }
            serverLauncherEnv_ = config.get("/tars/application/server<env>", "");
            timeout_ = TC_Common::strto<int>(
                    config.get("/tars/application/server<hearttimeout>", "60"));
        }
        catch (const exception& e)
        {
            TLOGERROR("call updateFromDescriptor got exception: " << e.what());
            return -1;
        }
        return 0;
    }
};

struct ServerRuntime
{
    pid_t pid_{ -1 };
    time_t lastLaunchTime_{ 0 };
    time_t lastTermTime_{ VERY_BIG_TIME_VALUE };
    time_t lastKillTime_{ VERY_BIG_TIME_VALUE };
    std::map<string, time_t> heatBeatTimes_{};

    void resetWithoutLaunchTime()
    {
        pid_ = -1;
        lastTermTime_ = VERY_BIG_TIME_VALUE;
        lastKillTime_ = VERY_BIG_TIME_VALUE;
        heatBeatTimes_.clear();
    }
};

struct ServerObjectImp
{
public:
    static ServerObjectImp& instance()
    {
        static ServerObjectImp imp;
        return imp;
    }

    int _generateTemplateConf()
    {
        auto registryProxy = ProxyManger::instance().getRegistryProxy();
        if (!registryProxy)
        {
            auto message = std::string("get null RegistryProxy");
            notifyMessage("[alarm] " + message);
            TLOGERROR(message << std::endl);
            return -1;
        }

        ServerDescriptor descriptor;
        try
        {
            int res = registryProxy->getServerDescriptor(metadata_.serverApp_,
                    metadata_.serverName_, descriptor);
            if (res < 0)
            {
                auto message = std::string(
                        "call registryProxy->getServerDescriptor() unexpected result: ").append(
                        std::to_string(res));
                notifyMessage("[alarm] " + message);
                TLOGERROR(message << std::endl);
                return -1;
            }
            TLOGDEBUG("get getServerDescriptor" << descriptor.writeToJsonString() << std::endl);
        }
        catch (const std::exception& e)
        {
            auto message = std::string(
                    "call registryProxy->getServerDescriptor() catch exception :").append(e.what());
            notifyMessage("[alarm] " + message);
            TLOGERROR(message << std::endl);
            return -1;
        }

        return metadata_.updateFromDescriptor(descriptor);
    }

    void startServer()
    {
        lock_guard<std::mutex> lockGuard(mutex_);
        serverTarget = ServerTarget::Start;
        setting_ = tars::Active;
        runtime_.lastLaunchTime_ = 0;
    }

    void restartServer()
    {
        lock_guard<std::mutex> lockGuard(mutex_);
        serverTarget = ServerTarget::Restart;
        setting_ = tars::Active;
        runtime_.lastLaunchTime_ = 0;
        _doStop();
    }

    void stopServer()
    {
        lock_guard<std::mutex> lockGuard(mutex_);
        serverTarget = ServerTarget::Stop;
        setting_ = tars::Inactive;
        _doStop();
    }

    int addFile(const std::string& file, std::string result)
    {
        lock_guard<std::mutex> lockGuard(mutex_);
        string fileName = TC_File::extractFileName(file);
        TarsRemoteConfig remoteConfig{};
        remoteConfig.setConfigInfo(Application::getCommunicator(), ServerConfig::Config,
                metadata_.serverApp_, metadata_.serverName_,
                metadata_.serverBaseDir_, "");
        return remoteConfig.addConfig(fileName, result);
    }

    void keepAlive(const ServerInfo& serverInfo)
    {
        std::lock_guard<mutex> lockGuard(mutex_);
        runtime_.pid_ = serverInfo.pid;
        if (serverTarget == ServerTarget::Start)
        {
            if (present_ == tars::Activating)
            {
                updateServerState(setting_, tars::Active, runtime_.pid_);
            }
            present_ = tars::Active;
            auto now = TNOW;
            if (serverInfo.adapter.empty())
            {
                for (auto& item: runtime_.heatBeatTimes_)
                {
                    item.second = now;
                }
            }
            else
            {
                runtime_.heatBeatTimes_[serverInfo.adapter] = now;
            }
            return;
        }
        _doStop();
    }

    void keepActiving(const ServerInfo& serverInfo)
    {
        std::lock_guard<mutex> lockGuard(mutex_);
        runtime_.pid_ = serverInfo.pid;
        if (serverTarget == ServerTarget::Start)
        {
            if (present_ == tars::Active)
            {
                updateServerState(setting_, tars::Activating, serverInfo.pid);
            }
            present_ = tars::Activating;
            auto now = TNOW;
            if (serverInfo.adapter.empty())
            {
                for (auto& item: runtime_.heatBeatTimes_)
                {
                    item.second = now;
                }
            }
            else
            {
                runtime_.heatBeatTimes_[serverInfo.adapter] = now;
            }
            return;
        }
        _doStop();
    }

    void startBackgroundPatrol()
    {
        thread_ = std::thread([this]
        {
            for (size_t i = 0;; ++i)
            {
                if (patrolStop_)
                {
                    return;
                }
                std::this_thread::sleep_for(std::chrono::seconds(1));
                if (i % INACTIVE_CHECK_INTERVAL == 0)
                {
                    inactivateCheck();
                }
                if (i % ACTIVE_CHECK_INTERVAL == 0)
                {
                    activateCheck();
                }
                if (i % UPDATE_STATE_INTERVAL == 0)
                {
                    updateServerState(setting_, present_, runtime_.pid_);
                }
            }
        });
        thread_.detach();
    }

    void startForegroundPatrol()
    {
        thread_ = std::thread([this]
        {
            for (size_t i = 0;; ++i)
            {
                if (patrolStop_)
                {
                    return;
                }
                std::this_thread::sleep_for(std::chrono::seconds(1));
                if (i % ACTIVE_CHECK_INTERVAL == 0)
                {
                    overtimeCheck();
                }

                if (i % UPDATE_STATE_INTERVAL == 0)
                {
                    updateServerState(setting_, present_, runtime_.pid_);
                }
            }
        });
        thread_.detach();
    }

    int generateTemplateConf()
    {
        if (_generateTemplateConf() != 0)
        {
            return -1;
        }
        return ::setenv(container::ServerLauncherArgvEnvKey, metadata_.serverLauncherArgv_.c_str(),
                1);
    }

    LauncherSetting generateLauncherSetting()
    {
        LauncherSetting setting{};
        setting.file_ = metadata_.serverLauncherFile_;
        setting.workDir_ = metadata_.serverBaseDir_;
        setting.argv_ = TC_Common::sepstr<string>(metadata_.serverLauncherArgv_, " ");
        setting.redirect_ = metadata_.redirect_;
        return setting;
    }

    ~ServerObjectImp()
    {
        patrolStop_ = true;
    }

private:

    void _doStart()
    {
        if (runtime_.lastLaunchTime_ + LAUNCH_SERVER_INTERVAL_TIME > TNOW)
        {
            return;
        }
        bool activateOk = false;
        assert(setting_ == tars::Active);
        do
        {
            int res = generateTemplateConf();
            if (res != 0)
            {
                TLOGERROR("generate server template config file error" << std::endl);
                return;
            }
            auto setting = generateLauncherSetting();
            auto pid = Launcher::activate(setting);
            if (pid > 0)
            {
                time_t now = TNOW;
                runtime_.resetWithoutLaunchTime();
                runtime_.lastLaunchTime_ = now;
                runtime_.pid_ = pid;
                present_ = Activating;
                for (auto&& adapter: metadata_.adapters)
                {
                    runtime_.heatBeatTimes_[adapter] = now;
                }
                activateOk = true;
                break;
            }
        } while (false);

        if (!activateOk)
        {
            present_ = tars::ServerState::Inactive;
        }
        updateServerState(setting_, present_, runtime_.pid_);
    }

    void _doStop()
    {
        if (!processExist(runtime_.pid_))
        {
            present_ = tars::Inactive;
            runtime_.resetWithoutLaunchTime();
            updateServerState(setting_, present_, runtime_.pid_);
            return;
        }
        present_ = Deactivating;
        updateServerState(setting_, present_, runtime_.pid_);
        do
        {
            if (metadata_.serverType_ != SERVER_TYPE_NOT_TARS)
            {
                AdminFPrx pAdminPrx = ProxyManger::instance().getAdminProxy();
                if (!pAdminPrx)
                {
                    TLOGERROR("get null AdminPrx");
                }
                else
                {
                    try
                    {
                        pAdminPrx->async_shutdown(nullptr);
                        break;
                    }
                    catch (...)
                    {
                    }
                }
            }
            //some program need send signal twice;
            ::kill(runtime_.pid_, SIGTERM);
            ::kill(runtime_.pid_, SIGTERM);
        } while (false);
        runtime_.lastTermTime_ = TNOW;
    }

    void inactivateCheck()
    {
        std::lock_guard<mutex> lockGuard(mutex_);

        if (serverTarget != ServerTarget::Restart && serverTarget != ServerTarget::Stop)
        {
            return;
        }

        do
        {
            if (runtime_.pid_ == -1 && present_ == tars::Inactive)
            {
                break;
            }

            if (!processExist(runtime_.pid_))
            {
                runtime_.pid_ = -1;
                present_ = tars::Inactive;
                runtime_.resetWithoutLaunchTime();
                updateServerState(setting_, present_, runtime_.pid_);
                break;
            }

            auto now = TNOW;
            if (runtime_.lastKillTime_ + MAX_KILL_TIME <= now)
            {
                auto message = std::string(
                        "after sending a SIGKILL to the process, the process still exists");
                notifyMessage("[alarm] " + message);
                TLOGERROR(message << std::endl;);
                std::cout << message << std::endl;
                exit(-1);
            }

            if (runtime_.lastTermTime_ + MAX_TERM_TIME <= now)
            {
                ::kill(runtime_.pid_, SIGKILL);
                runtime_.lastKillTime_ = now;
                return;
            }
            return;
        } while (false);
        assert(runtime_.pid_ == -1);
        assert(present_ == tars::Inactive);
        if (serverTarget == ServerTarget::Restart)
        {
            serverTarget = ServerTarget::Start;
        }
    }

    void activateCheck()
    {
        std::lock_guard<mutex> lockGuard(mutex_);
        if (serverTarget != ServerTarget::Start)
        {
            return;
        }
        assert(serverTarget == ServerTarget::Start);
        assert(setting_ == tars::Active);
        time_t now = TNOW;
        if (present_ == Activating || present_ == Active)
        {
            if (!processExist(runtime_.pid_))
            {
                present_ = tars::Inactive;
                runtime_.resetWithoutLaunchTime();
                _doStart();
                return;
            }

            bool timeout = false;
            for (const auto& item: runtime_.heatBeatTimes_)
            {
                if (item.second + metadata_.timeout_ < now)
                {
                    timeout = true;
                }
            }
            if (timeout)
            {
                auto message = std::string("heartbeat overtime, will restart process");
                notifyMessage("[alarm] " + message);
                TLOGWARN(message);
                serverTarget = ServerTarget::Restart;
                _doStop();
                return;
            }
            return;
        }

        if (present_ == tars::Inactive)
        {
            present_ = tars::Inactive;
            runtime_.resetWithoutLaunchTime();
            _doStart();
            return;
        }

        if (present_ == tars::Deactivating)
        {
            TLOGDEBUG("In " << __FUNCTION__ << "__line__: " << __LINE__ << std::endl;);
            assert(false); //should not reach here;
            return;
        }
    }

    void overtimeCheck()
    {
        time_t now = TNOW;
        std::lock_guard<mutex> lockGuard(mutex_);
        bool timeout = false;
        for (const auto& item: runtime_.heatBeatTimes_)
        {
            if (item.second + metadata_.timeout_ < now)
            {
                timeout = true;
                break;
            }
        }

        if (timeout)
        {
            auto message = std::string("heartbeat overtime| ");
            notifyMessage("[alarm] " + message);
            TLOGWARN(message);
            return;
        }
    }

private:
    ServerObjectImp() = default;;

private:
    std::mutex mutex_;
    ServerTarget serverTarget{ ServerTarget::Start };
    ServerState setting_{ tars::Active };
    ServerState present_{ tars::Inactive };
    ServerRuntime runtime_{};
    ServerMetadata metadata_{};
    std::thread thread_;
    bool patrolStop_{ false };
};

int ServerObject::startServer(const std::string& application, const std::string& serverName,
        std::string& result)
{
    if (application != container::serverApp && serverName != container::serverName)
    {
        return -1;
    }
    result = "Success";
    ServerObjectImp::instance().startServer();
    return 0;
}

int ServerObject::stopServer(const std::string& application, const std::string& serverName,
        std::string& result)
{
    if (application != container::serverApp && serverName != container::serverName)
    {
        return -1;
    }
    result = "Success";
    ServerObjectImp::instance().stopServer();
    return 0;
}

int ServerObject::restartServer(const std::string& application, const std::string& serverName,
        std::string& result)
{
    if (application != container::serverApp && serverName != container::serverName)
    {
        return -1;
    }
    result = "Success";
    ServerObjectImp::instance().restartServer();
    return 0;
}

int
ServerObject::addFile(const std::string& application, const std::string& serverName,
        const std::string& file, std::string& result)
{
    if (application != container::serverApp && serverName != container::serverName)
    {
        return -1;
    }
    return ServerObjectImp::instance().addFile(file, result);
}

int ServerObject::notifyServer(const std::string& application, const std::string& serverName,
        const std::string& command,
        std::string& result)
{
    try
    {
        AdminFPrx pAdminPrx = ProxyManger::instance().getAdminProxy();
        result = pAdminPrx->notify(command);
    }
    catch (const exception& e)
    {
        result = "error" + string(e.what());
        return -1;
    }
    return 0;
}

void ServerObject::keepActiving(const ServerInfo& serverInfo)
{
    if (serverInfo.application != container::serverApp &&
        serverInfo.serverName != container::serverName)
    {
        return;
    }
    ServerObjectImp::instance().keepActiving(serverInfo);
}

void ServerObject::keepAlive(const ServerInfo& serverInfo)
{
    if (serverInfo.application != container::serverApp &&
        serverInfo.serverName != container::serverName)
    {
        return;
    }
    ServerObjectImp::instance().keepAlive(serverInfo);
}

void ServerObject::startBackgroundPatrol()
{
    ServerObjectImp::instance().startBackgroundPatrol();
}

void ServerObject::startForegroundPatrol()
{
    ServerObjectImp::instance().startForegroundPatrol();
}

int ServerObject::generateTemplateConf()
{
    return ServerObjectImp::instance().generateTemplateConf();
}

LauncherSetting ServerObject::generateLauncherSetting()
{
    return ServerObjectImp::instance().generateLauncherSetting();
}
