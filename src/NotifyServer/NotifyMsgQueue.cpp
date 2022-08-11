#include "NotifyMsgQueue.h"
#include "servant/Application.h"
#include "ESHelper.h"
#include "util/tc_timer.h"

static void buildESPushContent(const NotifyRecord& record, std::ostringstream& stream)
{
    stream << R"({"create": {}})" << "\n";
    auto str = record.writeToJsonString();
    str.resize(str.size() - 1);
    stream << str;
    stream << "," << R"("@timestamp")"<<":"<< TNOW << "}" << "\n";
}

void NotifyMsgQueue::init(const TC_Config& config)
{

	initLimit(config);

	_index = config.get("/tars/elk/index<notify>");
	if (_index.empty())
	{
		auto message = std::string("get empty index value");
		TLOG_ERROR(message << std::endl);
		throw std::runtime_error(message);
	}

	auto age = config.get("/tars/elk/age<notify>", "3d");
	const auto& pattern = _index;
	const auto& policy = _index;

	ESHelper::setAddressByTConfig(config);
	ESHelper::createESPolicy(policy, age);
	ESHelper::createESDataStreamTemplate(_index, pattern, policy);

	_timer.startTimer();
	start();
}

void NotifyMsgQueue::terminate()
{
	_terminate = true;

	TC_ThreadLock::Lock lock(*this);
	notifyAll();
}

void NotifyMsgQueue::add(const NotifyRecord& notifyRecord)
{
	_qMsg.push_back(notifyRecord);
}

void NotifyMsgQueue::run()
{
	while (!_terminate)
	{
		vector<NotifyRecord> vQData;
		do
		{
			NotifyRecord data;
			_qMsg.pop_front(data, -1);
			if (!checkLimit(data.app + "." + data.server))
			{
				TLOG_ERROR("limit fail|" << data.app << "." << data.server << "|" << data.podName << "|" << data.level << "|" << data.message
										<< endl);
				continue;
			}
			vQData.push_back(data);
		} while ((!_qMsg.empty()) && (vQData.size() < 500));
		writeToES(vQData);
	}
}

void NotifyMsgQueue::writeToES(const vector<NotifyRecord>& data)
{
	std::ostringstream stream;
	for (auto&& item: data)
	{
		buildESPushContent(item, stream);
	}
	auto context = std::make_shared<ESRequestContext>();
	context->uri = "/" + _index + "/_bulk";
	context->body = stream.str();
	ESHelper::post2ESWithRetry(&_timer, context);
}

void FreqLimit::initLimit(const TC_Config& conf)
{
	string limitConf = conf.get("/tars/server<notify_limit>", "300:10");
	vector<int> vi = TC_Common::sepstr<int>(limitConf, ":,|");
	if (vi.size() != 2)
	{
		_interval = 300;
		_count = 10;
	}
	else
	{
		_interval = (unsigned int)vi[0];
		_count = vi[1];
		if (_count <= 1)
		{
			_count = 1;
		}
	}
}

bool FreqLimit::checkLimit(const string& sServer)
{
	auto it = _limit.find(sServer);
	time_t t = TNOW;
	if (it != _limit.end())
	{
		if (t > _limit[sServer].t + _interval)
		{
			_limit[sServer].t = t;
			_limit[sServer].count = 1;
			return true;
		}
		else if (_limit[sServer].count >= _count)
		{
			return false;
		}
		else
		{
			_limit[sServer].count++;
			return true;
		}
	}
	else
	{
		LimitData ld{};
		ld.t = t;
		ld.count = 1;
		_limit[sServer] = ld;
		return true;
	}
}