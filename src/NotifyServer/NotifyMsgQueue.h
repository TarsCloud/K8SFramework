#ifndef _NOTIFY_MSG_QUEUE_H_
#define _NOTIFY_MSG_QUEUE_H_

#include "servant/NotifyF.h"
#include "util/tc_config.h"
#include "util/tc_singleton.h"
#include "util/tc_thread.h"
#include "util/tc_thread_queue.h"
#include "NotifyRecord.h"
#include "util/tc_timer.h"

using namespace tars;

class FreqLimit
{
public:
	struct LimitData
	{
		unsigned int t;
		int count;
	};

	void initLimit(const TC_Config& conf);

	// return true 表示检测通过，没有被频率限制，  false: 被频率限制了
	bool checkLimit(const string& sServer);

protected:
	unordered_map<string, LimitData> _limit;

	unsigned int _interval;
	int _count;
};

class NotifyMsgQueue : public TC_Singleton<NotifyMsgQueue>,
					   public TC_ThreadLock,
					   public TC_Thread,
					   public FreqLimit
{
public:
	NotifyMsgQueue() = default;

	~NotifyMsgQueue() override = default;

	void init(const TC_Config& config);

	/**
	 * add
	 */
	void add(const NotifyRecord& notifyRecord);

	/**
	 * stop
	 */
	void terminate();

	/**
	 * 写到ES
	 */
	void writeToES(const vector<NotifyRecord>& data);

protected:
	void run() override;

protected:
	bool _terminate{ false };
	string _index{};
	TC_ThreadQueue<NotifyRecord> _qMsg{};
	TC_Timer _timer{};
};

#endif // _NOTIFY_MSG_QUEUE_H_
