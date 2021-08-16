#pragma once

#include <sys/types.h>
#include <sys/wait.h>
#include <RegistryServer/NodeDescriptor.h>
#include "Node.h"
#include "../RegistryServer/Registry.h"
#include "util/tc_file.h"
#include "util/tc_config.h"
#include "servant/AdminF.h"
#include "servant/NodeF.h"
#include "servant/RemoteLogger.h"
#include "util.h"
#include "TimerTaskQueue.h"
#include "ContainerDetail.h"

using namespace tars;
using namespace std;

class ServerObject : public std::enable_shared_from_this<ServerObject> {
public:

private:

    enum ProcessStatus {
        Done,
        Processing,
        Error
    };

    struct MetaData {
    public:
        MetaData() :
                _sServerApp(container_detail::imageBindServerApp),
                _sServerName(container_detail::imageBindServerName),
                _sServerType(container_detail::imageBindServerType),
                _sServerBaseDir(container_detail::imageBindServerBinDir),
                _sServerConfFile(container_detail::imageBindServerConfFile),
                _sServerLauncherFile(container_detail::imageBindServerLauncherFile),
                _sStdout_Stderr(container_detail::imageBindServerLogDir + "/" + _sServerApp + "/" + _sServerName + "/" + "stdout_stderr"),
                _sServerLauncherArgv(container_detail::imageBindServerLauncherArgv) {
        }

        const string &_sServerApp;
        const string &_sServerName;
        const string &_sServerType;
        const string &_sServerBaseDir;
        const string &_sServerConfFile;
        const string &_sServerLauncherFile;
        const string _sStdout_Stderr;

        string _sServerLauncherArgv;
        string _sServerLauncherEnv;       //环境变量字符串
        int _iTimeout{60};                 //心跳超时时间
        uint16_t _adminPort{1000};
        vector<string> _vAdapter{};       //adapter

        int updateMetaDataFromDescriptor(const ServerDescriptor &descriptor);

    private:
        int updateConfFromDescriptor(const ServerDescriptor &descriptor);

        int loadConf(TC_Config &tConf);
    };

    struct RuntimeData {
        int64_t _pid{-1};
        ServerState settingState{Active};
        ServerState presentState{Inactive};
        time_t _tLastStartTime{0};
        vector<std::pair<string, time_t>> _vAdapterKeepTimes{};
    };

public:

    ServerObject() = default;

    ~ServerObject() = default;

    inline const std::string &getApplication() {
        return _metaData._sServerApp;
    }

    inline const std::string &getServerName() {
        return _metaData._sServerName;
    }

    inline void setPid(int pid) {
        lock_guard<std::mutex> lockGuard(_mutex);
        if (_runtimeData._pid != pid) {
            _runtimeData._pid = pid;
        }
    }

    int startServer(std::string &result);

    int stopServer(std::string &result);

    int restartServer(std::string &result);

    int notifyServer(const string &command, string &result);

    int addFile(string sFile, std::string &result);

    void updateKeepAliveTime(const string &adapter);

    void updateKeepActiving();

    bool checkStopped();

    void kill();

    void checkServerState(); //用于周期性检测状态是否符合预期

    void uploadStopStat(int stopStat);

    void updateServerState();  // 用于周期性的被动上报

    static void updateServerState(tars::ServerState settingState, ServerState presentState); //用于在状态变化时，主动上报;

private:
    //以_开头的函数都不加锁, 由调用方保证安全.

    void _kill();

    int _stopServer(std::string &result);

    int _startServer(std::string &result);

    ProcessStatus _doStop(std::string &result);

    void _waitingStop();

    ProcessStatus _doStart(std::string &result);;

    bool _checkStopped();

    bool _isAutoStart();

private:
    std::mutex _mutex;
    MetaData _metaData;
    RuntimeData _runtimeData;
};

typedef std::shared_ptr<ServerObject> ServerObjectPtr;

