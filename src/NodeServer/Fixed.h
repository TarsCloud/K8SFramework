
#pragma once

constexpr size_t STOPPED_CHECK_INTERVAL = 1;  /* 检测服务是否停止的间隔时间　单位:秒 */

constexpr size_t START_SERVER_INTERVAL_TIME = 45;    /* 启动同一个服务最小间隔时间　单位:秒 */

constexpr pid_t MIN_PID_VALUE = 2;  /* 最小的合法 pid 值 */

constexpr size_t MAX_DEACTIVATING_TIME = 10;   /* 服务处于 Deactivating　状态的最长时间 单位:秒 */

constexpr char FIXED_NODE_PROXY_NAME[] = "tars.tarsnode.ServerObj@tcp -h 127.0.0.1 -p 19386 -t 60000";;
constexpr char FIXED_QUERY_PROXY_NAME[] = "tars.tarsregistry.QueryObj@tcp -h tars-tarsregistry -p 17890";
constexpr char FIXED_REGISTRY_PROXY_NAME[] = "tars.tarsregistry.RegistryObj@tcp -h tars-tarsregistry -p 17891";
constexpr char FIXED_NOTIFY_PROXY_NAME[] = "tars.tarsnotify.NotifyObj";
