#pragma once

#include "servant/Application.h"

using namespace tars;

class ConfigServer : public Application
{
protected:
    /**
     * ooo 初始化, 只会进程调用一次
     */
    void initialize() final;

    /**
     * 析构, 每个进程都会调用一次
     */
    void destroyApp() final;
};
