#pragma once

#include <string>
#include "servant/QueryF.h"

/**
 * 对象查询接口类
 */

class QueryImp : public QueryF
{
public:
    /**
     * 构造函数
     */
    QueryImp() = default;

    /**
     * 初始化
     */
    void initialize() override
    {
    };

    /**
     ** 退出
     */
    void destroy() override
    {
    };

    /** 根据id获取对象
     *
     * @param id 对象名称
     *
     * @return  返回所有该对象的活动endpoint列表
     */
    vector<EndpointF> findObjectById(const std::string& id, CurrentPtr current) override;

    /**根据id获取所有对象,包括活动和非活动对象
     *
     * @param id         对象名称
     * @param activeEp   存活endpoint列表
     * @param inactiveEp 非存活endpoint列表
     * @return:  0-成功  others-失败
     */
    Int32
    findObjectById4Any(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp,
            CurrentPtr current) override;

    /** 根据id获取对象所有endpoint列表
     *
     * @param id         对象名称
     * @param activeEp   存活endpoint列表
     * @param inactiveEp 非存活endpoint列表
     * @return:  0-成功  others-失败
     */
    Int32 findObjectById4All(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp,
            CurrentPtr current) override;

    /** 根据id获取对象同组endpoint列表
    *
    * @param id         对象名称
    * @param activeEp   存活endpoint列表
    * @param inactiveEp 非存活endpoint列表
    * @return:  0-成功  others-失败
    */
    Int32 findObjectByIdInSameGroup(const std::string& id, vector<EndpointF>& activeEp, vector<EndpointF>& inactiveEp,
            CurrentPtr current) override;

    /** 根据id获取对象指定归属地的endpoint列表
     *
     * @param id         对象名称
     * @param activeEp   存活endpoint列表
     * @param inactiveEp 非存活endpoint列表
     * @return:  0-成功  others-失败
     */
    Int32 findObjectByIdInSameStation(const std::string& id, const std::string& sStation, vector<EndpointF>& activeEp,
            vector<EndpointF>& inactiveEp,
            CurrentPtr current) override;

    /** 根据id获取对象同set endpoint列表
    *
    * @param id         对象名称
    * @param setId      set全称,格式为setname.setarea.setgroup
    * @param activeEp   存活endpoint列表
    * @param inactiveEp 非存活endpoint列表
    * @return:  0-成功  others-失败
    */
    Int32 findObjectByIdInSameSet(const std::string& id, const std::string& setId, vector<EndpointF>& activeEp,
            vector<EndpointF>& inactiveEp,
            CurrentPtr current) override;
};
