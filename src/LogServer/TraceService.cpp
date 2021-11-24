#include "TraceService.h"
#include "ESHelper.h"
#include "ESIndex.h"
#include "util/tc_file.h"
#include "ESWriter.h"
#include "TimerTaskQueue.h"
#include "servant/RemoteLogger.h"
#include "SnapshotDraftsman.h"

void TraceService::initialize(const TC_Config& config)
{
    string message{};
    if (!loadTimerValue(config, message))
    {
        throw std::runtime_error("bad config: " + message);
    }

    logDir_ = config.get("/tars/trace<log_dir>", "/usr/local/app/tars/remote_app_log/_tars_/_trace_");
    SnapshotDraftsman::setSavePath(ServerConfig::DataPath);

    ESHelper::setAddressByTConfig(config);

    timerThread_ = std::thread([]
    {
        TimerTaskQueue::instance().run();
    });
    timerThread_.detach();

    TimerTaskQueue::instance().pushCycleTask(
        [this](const size_t&, size_t&)
        {
            auto todayLogFile = buildLogFileName(false);
            auto absTodayLogFile = buildLogFileName(true);
            if (TC_File::isFileExist(absTodayLogFile))
            {
                onModify(todayLogFile);
            }
        }, 0, 1);

    ESWriter::createIndexTemplate();
};

bool TraceService::loadTimerValue(const TC_Config& config, string& message)
{
    snapshotTimer_ = TC_Common::strto<int>(config.get("/tars/trace<SnapshotTimer>", "300"));
    firstCheckTimer_ = TC_Common::strto<int>(config.get("/tars/trace<FirstCheckTimer>", "3"));
    checkCycleTimer_ = TC_Common::strto<int>(config.get("/tars/trace<CheckCycleTimer>", "3"));
    closureOvertime_ = TC_Common::strto<int>(config.get("/tars/trace<OvertimeWhenClosure>", "3"));
    maxOvertime_ = TC_Common::strto<int>(config.get("/tars/trace<Overtime>", "100"));

    if (snapshotTimer_ < 60 || snapshotTimer_ > 10000)
    {
        message = "SnapshotTimer value should in [60,10000]";
        return false;
    }

    if (firstCheckTimer_ < 1 || firstCheckTimer_ > 600)
    {
        message = "FirstCheckTimer value should in [1,600]";
        return false;
    }

    if (checkCycleTimer_ < 1 || checkCycleTimer_ > 100)
    {
        message = "CheckCycleTimer value should in [1,100]";
        return false;
    }

    if (closureOvertime_ < 1 || closureOvertime_ > 100)
    {
        message = "OvertimeWhenClosure value should in [1,100]";
        return false;
    }

    if (maxOvertime_ < 100 || maxOvertime_ > 600)
    {
        message = "Overtime value should in [100,600]";
        return false;
    }

    for (auto& pair: pairs_)
    {
        pair.second->aggregation->setTimer(firstCheckTimer_, checkCycleTimer_, closureOvertime_, maxOvertime_);
    }

    message = "done";
    return true;
}

void TraceService::onModify(const string& file)
{
    auto iterator = pairs_.find(file);
    if (iterator != pairs_.end())
    {
        auto pair = iterator->second;
        while (true)
        {
            auto&& rs = pair->logReader->read();
            if (rs.empty())
            {
                break;
            }
            TLOGDEBUG("read " << rs.size() << " lines " << endl);
            ESWriter::postRawLog(file, rs);
            for (auto&& r: rs)
            {
                pair->aggregation->pushRawLog(r);
            }
        }
        return;
    }

    if (isExpectedFile(file))
    {
        auto absFileLog = getAbsLogFileName(file);
        auto pair = std::make_shared<LogReadAggregationPair>(absFileLog);
        SnapshotDraftsman::restoreSnapshot(pair->logReader, pair->aggregation);

        std::weak_ptr<LogReadAggregationPair> weak_ptr = pair;
        TimerTaskQueue::instance().pushCycleTask([weak_ptr](const size_t&, size_t& nextTimer)
        {
            auto ptr = weak_ptr.lock();
            if (ptr != nullptr)
            {
                SnapshotDraftsman::createSnapshot(ptr->logReader, ptr->aggregation);
                return;
            }
            nextTimer = 0;
        }, snapshotTimer_, snapshotTimer_);
        pair->aggregation->setTimer(firstCheckTimer_, checkCycleTimer_, closureOvertime_, maxOvertime_);
        pairs_[file] = pair;
        while (true)
        {
            auto&& rs = pair->logReader->read();
            if (rs.empty())
            {
                break;
            }
            TLOGDEBUG("read " << rs.size() << " lines " << endl);
            ESWriter::postRawLog(file, rs);
            for (auto&& r: rs)
            {
                pair->aggregation->pushRawLog(r);
            }
        }

        /*
            we only monitor one file at the same time,
            if the current file is expected,means that other LogReadAggregationPair need to be deleted.
        */
        for (auto it = pairs_.begin(); it != pairs_.end();)
        {
            if (it->first == file)
            {
                it++;
                continue;
            }
            auto ptr = it->second;
            TimerTaskQueue::instance().pushTimerTask([ptr]
            {
                /*
                 *  ptr will release after 30 minutes.
                 */
            }, 60 * 30);
            pairs_.erase(it++);
        }
    }
}
