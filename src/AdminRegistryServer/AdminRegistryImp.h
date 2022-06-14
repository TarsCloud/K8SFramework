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

#ifndef __AMIN_REGISTRY_H__
#define __AMIN_REGISTRY_H__

#include "AdminReg.h"

using namespace tars;

/**
 * 管理控制接口类
 */
class AdminRegistryImp: public AdminReg
{
public:
    /**
     * 构造函数
     */
    AdminRegistryImp(){};

    /**
     * 初始化
     */
    virtual void initialize();

    /**
     ** 退出
     */
    virtual void destroy() {};

public:

    /**
     * ping node
     *
     * @param name: node id
     * @param out result : 结果描述
     *
     * @return : true-ping通；false-不通
     */
    virtual bool pingNode(const string & name, string &result, CurrentPtr current);

    /**
     * 启动特定server
     *
     * @param application: 应用
     * @param serverName : server名
     * @param nodeName   : node id
     * @param out result : 结果描述
     *
     * @return : 0-成功 others-失败
     */
    virtual int startServer(const string & application, const string & serverName, const string & nodeName,
            string &result, CurrentPtr current);

    /**
     * 停止特定server
     *
     * @param application: 应用
     * @param serverName : server名
     * @param nodeName   : node id
     * @param out result : 结果描述
     *
     * @return : 0-成功 others-失败
     */
    virtual int stopServer(const string & application, const string & serverName, const string & nodeName,
            string &result, CurrentPtr current);
    /**
     * 重启特定server
     *
     * @param application: 应用
     * @param serverName : server名
     * @param nodeName   : node id
     * @param out result : 结果描述
     *
     * @return : 0-成功 others-失败
     */
    virtual int restartServer(const string & application, const string & serverName, const string & nodeName,
            string &result, CurrentPtr current);
    /**
     * 通知服务
     * @param application
     * @param serverName
     * @param nodeName
     * @param command
     * @param result
     * @param current
     *
     * @return int
     */
    virtual int notifyServer(const string & application, const string & serverName, const string & nodeName,
            const string &command, string &result, CurrentPtr current);


	/**
	 * 注册插件
	 * @param conf
	 * @param current
	 * @return
	 */
	virtual int registerPlugin(const PluginConf &conf, CurrentPtr current);

	/**
	 * 是否有有开发权限
	 *
	 * @return : 返回值详见tarsErrCode枚举值
	 */
	virtual int hasDevAuth(const string &application, const string & serverName, const string & uid, bool &has, CurrentPtr current);

	/**
	 * 是否有运维权限
	 *
	 * @return : 返回值详见tarsErrCode枚举值
	 */
	virtual int hasOpeAuth(const string & application, const string & serverName, const string & uid, bool &has, CurrentPtr current);

	/**
	 * 是否有有管理员权限
	 *
	 * @return : 返回值详见tarsErrCode枚举值
	 */
	virtual int hasAdminAuth(const string & uid, bool &has, CurrentPtr current);

	/**
	 * 解析ticket, uid不为空则有效, 否则无效需要重新登录
	 *
	 * @return : 返回值详见tarsErrCode枚举值
	 */
	virtual int checkTicket(const string & ticket, string &uid, CurrentPtr current);

protected:
	string nodeNameToDns(const string &nodeName);
};


#endif
