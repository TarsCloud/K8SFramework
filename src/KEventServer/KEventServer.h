#pragma once

#include "servant/Application.h"

class KEventServer : public Application
{
 protected:

    /**
     * 析构, 每个进程都会调用一次
     */
    void destroyApp() override;

 public:
/**
 * 初始化, 只会进程调用一次
 */
    void initialize() override;
};
