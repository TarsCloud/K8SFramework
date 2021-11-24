#pragma once

#include "LogReader.h"
#include "LogAggregation.h"
#include "util/tc_config.h"
#include "util/tc_common.h"

class TraceService
{
 protected:
    struct LogReadAggregationPair : public std::enable_shared_from_this<LogReadAggregationPair>
    {
        explicit LogReadAggregationPair(const std::string& logFile)
        {
            logReader = std::make_shared<LogReader>(logFile);
            aggregation = std::make_shared<LogAggregation>(logFile);
        };
        std::shared_ptr<LogReader> logReader;
        std::shared_ptr<LogAggregation> aggregation{};
    };

 public:
    TraceService() = default;

    bool loadTimerValue(const tars::TC_Config& config, std::string& message);

    void initialize(const tars::TC_Config& config);

 private:

    void onModify(const std::string& file);

    bool isExpectedFile(const std::string& file)
    {
        return file == buildLogFileName(false);
    }

    std::string buildLogFileName(bool abs)
    {
        return (abs ? (logDir_ + "/") : "") + logPrefix_ + tars::TC_Common::nowdate2str() + ".log";
    }

    std::string getAbsLogFileName(const std::string& name)
    {
        return logDir_ + "/" + name;
    }

 private:
    std::string logDir_{};
    std::string logPrefix_{ "_tars_._trace___t_trace__" };
    std::thread timerThread_;
    std::map<std::string, std::shared_ptr<LogReadAggregationPair>> pairs_;

    size_t snapshotTimer_{ 300 };
    size_t firstCheckTimer_{ 3 };
    size_t checkCycleTimer_{ 3 };
    size_t closureOvertime_{ 3 };
    size_t maxOvertime_{ 100 };
};
