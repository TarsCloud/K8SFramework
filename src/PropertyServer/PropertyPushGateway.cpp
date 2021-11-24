#include "PropertyPushGateway.h"
#include "JsonTr.h"
#include "ESHelper.h"
#include "util/tc_timer.h"

static void
buildESPostContent(const string& sDate, const string& sTFlag, const StatPropMsgHead& head, const StatPropMsgBody& body, std::ostringstream& stream)
{
	for (const auto& item: body.vInfo)
	{
		stream << R"({"index": {}})" << "\n";
		stream << "{";
		stream << jsonTr("f_timestamp") << ":" << jsonTr(TNOW);
		stream << "," << jsonTr("f_date") << ":" << jsonTr(sDate);
		stream << "," << jsonTr("f_tflag") << ":" << jsonTr(sTFlag);
		stream << "," << jsonTr("master_name") << ":" << jsonTr(head.moduleName);
		stream << "," << jsonTr("master_ip") << ":" << jsonTr(head.ip);
		stream << "," << jsonTr("property_name") << ":" << jsonTr(head.propertyName);
		stream << "," << jsonTr("set_name") << ":" << jsonTr(head.setName);
		stream << "," << jsonTr("set_area") << ":" << jsonTr(head.setArea);
		stream << "," << jsonTr("set_id") << ":" << jsonTr(head.setID);
		stream << "," << jsonTr("policy") << ":" << jsonTr(item.policy);
		if (item.policy == "Avg")
		{
			vector<string> sTmp = TC_Common::sepstr<string>(item.value, "=");
			double avg{};
			if (2 == sTmp.size() && TC_Common::strto<long>(sTmp[1]) != 0)
			{
				avg = TC_Common::strto<double>(sTmp[0]) / TC_Common::strto<double>(sTmp[1]);
			}
			else
			{
				avg = TC_Common::strto<double>(sTmp[0]);
			}
			stream << "," << jsonTr("value") << ":" << jsonTr(avg);
		}
		else if (item.policy != "Distr")
		{
			stream << "," << jsonTr("value") << ":" << jsonTr(TC_Common::strto<double>(item.value));
		}
		else
		{
			stream << "," << jsonTr("value") << ":" << jsonTr(0);
		}
		stream << "}";
		stream << "\n";
	}
}

void PropertyPushGateway::push(const tars::StatPropMsgHead& head, const tars::StatPropMsgBody& body)
{
	std::lock_guard<std::mutex> lockGuard(mutex);
	assert(_cachePtr == _cache[0] || _cachePtr == _cache[1]);
	_cachePtr->add(head, body);
}

void PropertyPushGateway::init(const TC_Config& config)
{

	_indexPre = config.get("/tars/es/indexpre<property>");
	if (_indexPre.empty())
	{
		auto message = std::string("get empty index value");
		TLOGERROR(message << std::endl);
		throw std::runtime_error(message);
	}


	auto age = config.get("/tars/es/age<property>", "15d");

	auto pattern = _indexPre + '*';
	auto policy = _indexPre;

	ESHelper::setAddressByTConfig(config);
	ESHelper::createESPolicy(policy, age);
	ESHelper::createESIndexTemplate(_indexPre, pattern, policy);

	initCache(config);
	_cachePtr = _cache[0];
}

void PropertyPushGateway::start()
{
	updateNextSyncFlag();
	timer_.postRepeated(1000 * 7, false,
			[this]()
			{
				if (isSyncTime())
				{
					sync();
					updateNextSyncFlag();
				}
			});
	timer_.startTimer();
}

void PropertyPushGateway::sync()
{

	PropertyHashMap* willSyncCachePtr = _cachePtr;
	{
		std::lock_guard<std::mutex> lockGuard(_mutex);
		assert(_cachePtr == _cache[0] || _cachePtr == _cache[1]);
		if (_cachePtr == _cache[0])
		{
			_cachePtr = _cache[1];
		}
		else if (_cachePtr == _cache[1])
		{
			_cachePtr = _cache[0];
		}

		_cachePtr->clear();
	}


	std::string index = _indexPre + _date;
	char tflag[4 + 1] = {};
	sprintf(tflag, "%.4lu", _nextSyncTFlag);

	std::ostringstream stream{};
	auto count = 0;
	for (auto item: *willSyncCachePtr)
	{
		StatPropMsgHead head{};
		StatPropMsgBody body{};
		int ret = item.get(head, body);
		if (ret == 0)
		{
			buildESPostContent(_date, tflag, head, body, stream);
		}
		if (count > 1 && count % 1024 == 0)
		{
			auto requestContext = std::make_shared<ESRequestContext>();
			requestContext->uri = "/" + index + "/_bulk";
			requestContext->body = stream.str();
			ESHelper::post2ESWithRetry(&timer_, requestContext);
			stream.str("");
		}
		++count;
	}
	auto requestContext = std::make_shared<ESRequestContext>();
	requestContext->uri = "/" + index + "/_bulk";
	requestContext->body = stream.str();
	ESHelper::post2ESWithRetry(&timer_, requestContext);
	willSyncCachePtr->clear();
}

bool PropertyPushGateway::isSyncTime() const
{
	size_t currentSyncFlag;
	time_t t = TNOW;
	struct tm ptm{};
	localtime_r(&t, &ptm);
	if (ptm.tm_min == 0)
	{
		if (ptm.tm_hour == 0)
		{
			currentSyncFlag = 2360;
		}
		else
		{
			currentSyncFlag = (ptm.tm_hour - 1) * 100 + 60;
		}
	}
	else
	{
		currentSyncFlag = ptm.tm_hour * 100 + ptm.tm_min;
	}
	return currentSyncFlag >= _nextSyncTFlag;
}

void PropertyPushGateway::updateNextSyncFlag()
{
	time_t t = TNOW;
	struct tm tm{};
	localtime_r(&t, &tm);
	_nextSyncTFlag = tm.tm_hour * 100 + tm.tm_min + 1;
	sprintf(_date, "%.4d%.2d%.2d", tm.tm_year + 1900, tm.tm_mon + 1, tm.tm_mday);
}

void PropertyPushGateway::initCache(const TC_Config& config)
{
	TLOGDEBUG("PropertyServer::initHashMap begin" << endl);
	for (auto& k: _cache)
	{
		k = new PropertyHashMap();
	}

	auto iMinBlock = TC_Common::strto<int>(config.get("/tars/hashmap<minBlock>", "128"));
	auto iMaxBlock = TC_Common::strto<int>(config.get("/tars/hashmap<maxBlock>", "256"));
	auto iFactor = TC_Common::strto<float>(config.get("/tars/hashmap<factor>", "2"));
	auto iSize = TC_Common::toSize(config.get("/tars/hashmap<size>"), 1024 * 1024 * 256);

	TLOGDEBUG("PropertyServer::initHashMap init multi hashmap begin..." << endl);

	for (int i = 0; i < 2; ++i)
	{
		string sHashMapFile = ServerConfig::DataPath + "/" + config.get("/tars/hashmap<file>", "hashmap.dat");

		string sPath = TC_File::extractFilePath(sHashMapFile);

		if (!TC_File::makeDirRecursive(sPath))
		{
			TLOGERROR("cannot create hashmap file " << sPath << endl);
			exit(0);
		}

		try
		{
			_cache[i]->initDataBlockSize(iMinBlock, iMaxBlock, iFactor);

			key_t key = tars::hash<string>()(TC_Common::tostr(i).append(ServerConfig::LocalIp).append("-").append(sHashMapFile));

			RemoteNotify::getInstance()->report("shm key:" + TC_Common::tostr((uint32_t)key) + ", size:" + TC_Common::tostr(iSize), false);

			TLOGINFO("initDataBlockSize size: " << iMinBlock << ", " << iMaxBlock << ", " << iFactor << ", key: " << key << endl);

			_cache[i]->initStore(key, iSize);

			TLOGINFO("\n" << _cache[i]->desc() << endl);
		}
		catch (TC_HashMap_Exception& e)
		{
			RemoteNotify::getInstance()->report(string("init error: ") + e.what(), false);

			TC_Common::msleep(100);

			TC_File::removeFile(sHashMapFile, false);
			throw runtime_error(e.what());
		}
	}
	TLOGDEBUG("PropertyServer::initHashMap init multi hashmap end..." << endl);
}
