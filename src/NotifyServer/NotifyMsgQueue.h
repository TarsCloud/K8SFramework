#ifndef _NOTIFY_MSG_QUEUE_H_
#define _NOTIFY_MSG_QUEUE_H_

#include "servant/NotifyF.h"
#include "util/tc_common.h"
#include "util/tc_config.h"
#include "util/tc_monitor.h"
// #include "util/tc_mysql.h"
#include "util/tc_singleton.h"
#include "util/tc_thread.h"
#include "util/tc_thread_queue.h"
#include "NotifyRecord.h"
#include "util/tc_http_async.h"

using namespace tars;
class FreqLimit 
{
public:
    struct LimitData 
    {
        unsigned int t;
        int count;
    };

    void initLimit(TC_Config *conf);

    // return true 表示检测通过，没有被频率限制，  false: 被频率限制了
    bool checkLimit(const string &sServer);

protected:
    unordered_map<string, LimitData> _limit;

    unsigned int _interval;
    int _count;
};

class NotifyMsgQueue : public TC_Singleton<NotifyMsgQueue>,
                       public TC_ThreadLock,
                       public TC_Thread,
                       public FreqLimit {
public:
    NotifyMsgQueue() = default;

    ~NotifyMsgQueue() = default;

    void init();

    /**
     * add
     */ 
    void add(const NotifyRecord &notifyRecord);

    /**
     * stop
     */
    void terminate();

    /**
     * 写到ES
     */ 
    void writeToES(const vector<NotifyRecord> &data, const string &date);


protected:
    virtual void run();

    void initElkTupleNodes(const tars::TC_Config &config);

    string getELKNodeAddress();

    string writeToJson(const NotifyRecord& record);

protected:
    bool _terminate;

    TC_HttpAsync                    _ast;

    TC_ThreadQueue<NotifyRecord>    _qMsg;

    string _protocol;
    vector<tuple<string, int>> _elkTupleNodes;

    string _indexPre;
};

#endif // _NOTIFY_MSG_QUEUE_H_
