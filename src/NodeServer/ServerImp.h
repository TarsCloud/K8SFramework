
#pragma once

#include "Node.h"
#include "servant/NodeF.h"

using namespace tars;
using namespace std;

class ServerImp : public ServerF
{
public:
    /**
     *
     */
    ServerImp() = default;;

    /**
     * 销毁服务
     * @param k
     * @param v
     *
     * @return int
     */
    ~ServerImp() override = default;

    /**
    * 初始化
    */
    void initialize() override
    {
    };

    /**
    * 退出
    */
    void destroy() override
    {
    };

    /**
    * 退出
    */

    int keepActiving(const tars::ServerInfo& serverInfo, TarsCurrentPtr current) override;

    int keepAlive(const tars::ServerInfo& serverInfo, TarsCurrentPtr current) override;

    int reportVersion(const std::string& app, const std::string& serverName, const std::string& version,
            TarsCurrentPtr current) override;

    tars::UInt32 getLatestKeepAliveTime(tars::CurrentPtr current) override;
};
