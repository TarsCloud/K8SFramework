﻿/**
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

#include "servant/Application.h"
#include "util/tc_http_async.h"

using namespace tars;

/**
 *  AdminRegistryServer Server
 */
class AdminRegistryServer : public Application
{
protected:
    /**
     * 初始化, 只会进程调用一次
     */
    virtual void initialize();

    /**
     * 析构, 每个进程都会调用一次
     */
    virtual void destroyApp();

public:
	TC_HttpAsync _ast;

};

extern AdminRegistryServer g_app;
