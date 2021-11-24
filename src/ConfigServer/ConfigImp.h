#ifndef _CONFIG_IMP_H_
#define _CONFIG_IMP_H_

#include "servant/ConfigF.h"

using namespace tars;

class ConfigImp : public Config
{
 public:
    /**
     *
     */
    ConfigImp() = default;

    /**
     *
     */
    ~ConfigImp() override = default;

    /**
     * 初始化
     *
     * @return int
     */
    void initialize() override;

    /**
     * 退出
     */
    void destroy() override
    {
    };

    /**
    * 获取配置文件列表
    * param app :应用
    * param server:  server名
    * param vf: 配置文件名
    *
    * return  : 配置文件内容
    */
    int ListConfig(const string& app, const string& server, vector<string>& vf, CurrentPtr current) override;

    /**
     * 加载配置文件
     * param app :应用
     * param server:  server名
     * param filename:  配置文件名
     *
     * return  : 配置文件内容
     */
    int loadConfig(const std::string& app, const std::string& server, const std::string& filename, string& config, CurrentPtr current) override;

    /**
     * 根据ip获取配置
     * @param appServerName
     * @param filename
     * @param host
     * @param config
     *
     * @return int
     */
    int loadConfigByHost(const string& appServerName, const string& filename, const string& host, string& config, CurrentPtr current) override;

    /**
     *
     * @param appServerName
     * @param filename
     * @param host
     * @param current
     *
     * @return int
     */
    int checkConfig(const string& appServerName, const string& filename, const string& host, string& result, CurrentPtr current) override;

    /**
    * 获取配置文件列表
    * param configInfo ConfigInfo
    * param vf: 配置文件名
    *
    * return  : 配置文件内容
    */
    int ListConfigByInfo(const ConfigInfo& configInfo, vector<string>& vf, CurrentPtr current) override;

    /**
     * 加载配置文件
     * param configInfo ConfigInfo
     * param config:  配置文件内容
     *
     * return  :
     */

    int loadConfigByInfo(const ConfigInfo& configInfo, string& config, CurrentPtr current) override;

    /**
     *
     * @param configInfo ConfigInfo
     *
     * @return int
     */

    int checkConfigByInfo(const ConfigInfo& configInfo, string& result, CurrentPtr current) override;

    /**
    * 获取服务的所有配置文件列表，
    * @param configInfo 支持拉取应用配置列表，服务配置列表，机器配置列表和容器配置列表
    * @param[out] vf  获取到的文件名称列表
    * @return int 0: 成功, -1:失败
    **/
    Int32 ListAllConfigByInfo(const GetConfigListInfo& configInfo, vector<std::string>& vf, CurrentPtr current) override;
};

#endif

