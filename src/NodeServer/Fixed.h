
#pragma once

constexpr ssize_t UPDATE_STATE_INTERVAL = 420; /* 上传服务状态间隔时间, 单位:秒 */
constexpr ssize_t ACTIVE_CHECK_INTERVAL = 1;  /* 检测服务是否启动的间隔时间, 单位:秒 */
constexpr ssize_t INACTIVE_CHECK_INTERVAL = 1;  /* 检测服务是否停止的间隔时间, 单位:秒 */
constexpr ssize_t LAUNCH_SERVER_INTERVAL_TIME = 30;    /* 启动服务间隔时间, 单位:秒 */

constexpr pid_t MIN_PID_VALUE = 2;    /* 最小的合法 pid 值 */

constexpr ssize_t MAX_TERM_TIME = 10;  /* 向业务进程发送 AminObj->async_shutdown() 或 发送 SIGTERM信号后的等待时间,若超时后Pid仍然存在,则发送SIGKILL信号, 单位:秒 */

constexpr ssize_t MAX_KILL_TIME = 3;   /* 向业务进程发送 SIGKILL 信号后的等待时间, 若超时后Pid仍然存在,则tarsnode退出, 单位:秒 */

constexpr char FIXED_NODE_PROXY_NAME[] = "tars.tarsnode.ServerObj@tcp -h 127.0.0.1 -p 19386 -t 60000";;
constexpr char FIXED_QUERY_PROXY_NAME[] = "tars.tarsregistry.QueryObj@tcp -h tars-tarsregistry -p 17890";
constexpr char FIXED_REGISTRY_PROXY_NAME[] = "tars.tarsregistry.RegistryObj@tcp -h tars-tarsregistry -p 17891";
constexpr char FIXED_LOCAL_ENDPOINT[] = "tcp -h 127.0.0.1 -p 1000 -t 3000";
constexpr char FIXED_LOCAL_PROXY_NAME[] = "AdminObj@tcp -h 127.0.0.1 -p 1000 -t 3000";
constexpr char FIXED_NOTIFY_PROXY_NAME[] = "tars.tarsnotify.NotifyObj";

constexpr char SERVER_TYPE_TARS_CPP[] = "cpp";
constexpr char SERVER_TYPE_TARS_NODE[] = "nodejs";
constexpr char SERVER_TYPE_TARS_PHP[] = "php";
constexpr char SERVER_TYPE_TARS_NODE_PKG[] = "nodejs-pkg";
constexpr char SERVER_TYPE_TARS_JAVA_WAR[] = "java-war";
constexpr char SERVER_TYPE_TARS_JAVA_JAR[] = "java-jar";
constexpr char SERVER_TYPE_NOT_TARS[] = "not-tars";

constexpr char SERVER_BACKGROUND_LAUNCH[] = "background";
constexpr char SERVER_FOREGROUND_LAUNCH[] = "foreground";
constexpr char SERVER_FOREGROUND_ADMIN_PORT[] = "19385";

constexpr char TARSNODE_CONFIG_TARGET[] = "config";
constexpr char TARSNODE_DAEMON_TARGET[] = "daemon";
