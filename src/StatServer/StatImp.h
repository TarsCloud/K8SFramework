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

#ifndef __STAT_IMP_H_
#define __STAT_IMP_H_

#include <functional>
#include "util/tc_common.h"
#include "util/tc_thread.h"
#include "util/tc_option.h"
#include "util/tc_file.h"
#include "util/tc_mysql.h"
#include "util/tc_config.h"
#include "util/tc_hash_fun.h"
#include "servant/RemoteLogger.h"
#include "jmem/jmem_hashmap.h"
#include "servant/StatF.h"


using namespace tars;

class StatImp : public StatF {
public:
    /**
     *
     */
    StatImp()=default;

    ~StatImp() override = default;

    /**
     * 初始化
     *
     * @return int
     */
    void initialize() override;

    /**
     * 退出
     */
    void destroy() override {
    };

    /**
     * 上报模块间调用信息
     * @param statmsg, 上报信息
     * @return int, 返回0表示成功
     */
    int reportMicMsg(const map<tars::StatMicMsgHead, tars::StatMicMsgBody> &statmsg, bool bFromClient,
                             tars::CurrentPtr current) override;

    /**
     * 上报模块间调用采样信息
     * @param sample, 上报信息
     * @return int, 返回0表示成功
     */
    int reportSampleMsg(const vector<StatSampleMsg> &msg, tars::CurrentPtr current) override;

private:
    string getSlaveName(const string &sSlaveName);
};

#endif


