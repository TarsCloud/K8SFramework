
#ifndef __LOG_SERVER_H_
#define __LOG_SERVER_H_

#include "servant/Application.h"

using namespace tars;

class LogServer : public Application
{
protected:
    /**
     * 初始化, 进程会调用一次
     */
    virtual void initialize();

    /**
     * 析构, 进程退出时会调用一次
     */
    virtual void destroyApp();

private:

    bool loadLogFormat(const string& command, const string& params, string& result);
};

#endif

