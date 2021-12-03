/**
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

#include "QueryImp.h"
#include "util/tc_http.h"
#include "util/tc_json.h"
#include "ESClient.h"

using namespace std;

static std::function<std::string()> ESIndexPre{};

//sqlPart:  [dataid]=[tars_stat]   [method]=[query] [date1]=[20200304]  [date2]=[20200304]
// [groupCond]=[ group by f_tflag]  [groupField]=[f_tflag]
// [sumField]=[ sum(succ_count),  sum(timeout_count),  sum(exce_count),  sum(total_time)]
// [tflag1]=[0000]  [tflag2]=[2360]  [uid]=[5|]
// [whereCond]=[ where slave_name like 'tars.tarsstat' and f_date='20200304' and f_tflag>='0000' and f_tflag<='2360' and slave_name like 'tars.tarsstat']

struct QueryParams
{
    QueryParams(std::size_t _sumFiledSize, std::size_t _groupFiledSize)
        : sumFiledSize(_sumFiledSize), groupFiledSize(_groupFiledSize)
    {
    }

    const std::size_t sumFiledSize;
    const std::size_t groupFiledSize;
    std::string queryUrl;
    std::string queryBody;
    std::string queryCursor;
};

static void updateQueryParams(const tars::MonitorQueryReq& req, std::shared_ptr<QueryParams>& params)
{
    if (params != nullptr)
    {
        if (params->queryCursor.empty())
        {
            assert(false);
        }
        params->queryBody.clear();
        params->queryBody.append(R"({"cursor":")").append(params->queryCursor).append(R"("})");
        return;
    }

    assert(params == nullptr);
    params = std::make_shared<QueryParams>(req.indexs.size(), req.groupby.size());
    params->queryUrl = "/_sql";

    std::ostringstream queryBodyStream;
    queryBodyStream << R"({"query":")";
    queryBodyStream << "SELECT ";
    for (size_t i = 0; i < req.indexs.size(); ++i)
    {
        if (i != 0)
        {
            queryBodyStream << ",";
        }
        queryBodyStream << "ifnull(sum(" << req.indexs[i] << "),0)";
    }
    for (const auto& groupFiled: req.groupby)
    {
        queryBodyStream << "," << groupFiled;
    }

    queryBodyStream << " FROM \\\"" << ESIndexPre() << req.date << "\\\"";
    queryBodyStream << " WHERE 1=1 ";

    if (!req.tflag1.empty())
    {
        queryBodyStream << " AND f_tflag>='" << req.tflag1 << "'";
    }
    if (!req.tflag2.empty())
    {
        queryBodyStream << " AND f_tflag<='" << req.tflag2 << "'";
    }

    for (auto&& condition: req.conditions)
    {
        string op;
        switch (condition.op)
        {
        case EQ:
            op = "=";
            break;
        case GT:
            op = ">";
            break;
        case GTE:
            op = ">=";
            break;
        case LT:
            op = "<";
            break;
        case LTE:
            op = "<=";
            break;
        case LIKE:
            op = "LIKE";
            break;
        default:
            continue;
        }
        queryBodyStream << " AND " + condition.field + " " + op + " '" + condition.val + "'";
    }
    for (size_t i = 0; i < req.groupby.size(); ++i)
    {
        if (i == 0)
        {
            queryBodyStream << " GROUP BY f_tflag";
        }

        queryBodyStream << "," << req.groupby[i];
    }
    queryBodyStream << " ORDER BY f_tflag";
    queryBodyStream << R"("})";
    params->queryBody = queryBodyStream.str();
    TLOGDEBUG("sql:" << params->queryBody << std::endl);
}

static size_t
parserQueryResponse(const std::string& responseContent, const std::shared_ptr<QueryParams>& params, tars::MonitorQueryRsp& queryResponse)
{
    const std::size_t sumFiledBeginSeq = 0;
    const std::size_t sumFiledEndSeq = params->sumFiledSize;

    const std::size_t groupFiledBeginSeq = params->sumFiledSize;
    const std::size_t groupFiledEndSeq = params->sumFiledSize + params->groupFiledSize;

    auto jsonPtr = tars::TC_Json::getValue(responseContent);
    auto jsonValuePtr = tars::JsonValueObjPtr::dynamicCast(jsonPtr);

    const auto& jsonValue = jsonValuePtr->value;
    auto cursorPtrIterator = jsonValue.find("cursor");
    if (cursorPtrIterator == jsonValue.end())
    {
        params->queryCursor = "";
    }
    else
    {
        auto cursorPtr = cursorPtrIterator->second;
        auto cursorValuePtr = tars::JsonValueStringPtr::dynamicCast(cursorPtr);
        params->queryCursor = cursorValuePtr->value;
    }

    auto rowsPtr = jsonValuePtr->get("rows");
    auto rowsValuePtr = tars::JsonValueArrayPtr::dynamicCast(rowsPtr);

    const auto& rowsValue = rowsValuePtr->value;
    for (auto&& rowsValueItem: rowsValue)
    {
        auto rowsElemValuePtr = tars::JsonValueArrayPtr::dynamicCast(rowsValueItem);
        const auto& rowElemValue = rowsElemValuePtr->value;
        std::vector<double> sumValues{};
        for (auto i = sumFiledBeginSeq; i < sumFiledEndSeq; ++i)
        {
            if (rowElemValue[i]->getType() == tars::eJsonTypeNum)
            {
                sumValues.push_back(tars::JsonValueNumPtr::dynamicCast(rowElemValue[i])->value);
            }
            else
            {
                assert(false);
            }
        }
        std::ostringstream groupKeyStream;
        for (auto i = groupFiledBeginSeq; i < groupFiledEndSeq; ++i)
        {
            if (i != groupFiledBeginSeq)
            {
                groupKeyStream << ",";
            }
            if (rowElemValue[i]->getType() == tars::eJsonTypeString)
            {
                groupKeyStream << tars::JsonValueStringPtr::dynamicCast(rowElemValue[i])->value;
            }
            else if (rowElemValue[i]->getType() == tars::eJsonTypeNum)
            {
                groupKeyStream << tars::JsonValueNumPtr::dynamicCast(rowElemValue[i])->value;
            }
            else
            {
                assert(false);
            }
        }
        queryResponse.result[groupKeyStream.str()] = std::move(sumValues);
    }
    queryResponse.ret = 0;
    queryResponse.msg = "";

    TLOGDEBUG("rsp ret:" << queryResponse.ret << ", msg:" << queryResponse.msg << std::endl);

    return rowsValue.size();
}

void QueryImp::setIndexPre(const string& indexPre)
{
    ESIndexPre = [indexPre]()
    {
        return indexPre;
    };
}

void QueryImp::initialize()
{
}

int QueryImp::query(const tars::MonitorQueryReq& req, tars::MonitorQueryRsp& rsp, tars::TarsCurrentPtr current)
{
    auto params = std::shared_ptr<QueryParams>();
    while (true)
    {
        updateQueryParams(req, params);
        std::string response{};
        int res = ESClient::instance().doRequest(ESClientRequestMethod::Get, params->queryUrl, params->queryBody, response);
        if (res != 200)
        {
            TLOGERROR("do elk request error: " << response << std::endl);
            rsp.ret = -1;
            rsp.msg = response;
            return -1;
        }
        size_t rowSize = parserQueryResponse(response, params, rsp);
        if (rowSize <= 0 || params->queryCursor.empty())
        {
            return rsp.ret;
        }
    }
}

void QueryImp::destroy()
{
}
