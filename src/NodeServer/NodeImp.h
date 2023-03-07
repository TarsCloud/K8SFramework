#pragma once

#include "Node.h"
#include <string>

using namespace tars;
using namespace std;

class NodeImp : public Node
{
public:
    /**
     *
     */
    NodeImp() = default;

    /**
     * 销毁服务
     * @param k
     * @param v
     *
     * @return
     */
    ~NodeImp() override = default;

    /**
    * 初始化
    */
    void initialize() override;

    /**
    * 退出
    */
    void destroy() override
    {
    };

    /**
    * 加载指定文件
    * @param out result  失败说明
    * @return  int 0成功  非0失败
    */
    int addFile(const std::string& application, const std::string& serverName, const std::string& file, string& result,
            TarsCurrentPtr current) override;

    /**
    * 启动指定服务
    * @param application    服务所属应用名
    * @param serverName  服务名
    * @return  int
    */
    int startServer(const std::string& application, const std::string& serverName, string& result,
            TarsCurrentPtr current) override;

    /**
    * 停止指定服务
    * @param application    服务所属应用名
    * @param serverName  服务名
    * @return  int
    */
    int stopServer(const std::string& application, const std::string& serverName, string& result,
            TarsCurrentPtr current) override;

    /**
    * 重启指定服务
    * @param application    服务所属应用名
    * @param serverName  服务名
    * @param seconds  重启等待时间
    * @return  int
    */
    int restartServer(const std::string& application, const std::string& serverName, std::string& result,
            TarsCurrentPtr current) override;

    /**
     * 通知服务
     * @param application
     * @param serverName
     * @param result
     * @param current
     *
     * @return int
     */
    int notifyServer(const std::string& application, const std::string& serverName, const std::string& command,
            string& result, TarsCurrentPtr current) override;

};

