#include "ConfigImp.h"
#include "Storage.h"
#include "K8SInterface.h"
#include "servant/NotifyF.h"
#include "servant/RemoteLogger.h"
#include "util/tc_config.h"

using namespace tars;

void ConfigImp::initialize()
{
}

int ConfigImp::ListConfig(const std::string& app, const std::string& server, vector<string>& vf, CurrentPtr current)
{
    TLOGDEBUG("ListConfig|" << app << "." << server << "|" << current->getIp() << std::endl);
    try
    {
        K8SInterface::listConfig(app, server, current->getIp(), vf);
    }
    catch (const std::exception& e)
    {
        TLOGERROR(e.what() << endl);
        return -1;
    }
    return 0;
}

int ConfigImp::loadConfigByHost(const std::string& appServerName, const std::string& filename, const std::string& host,
        string& config, CurrentPtr current)
{
    TLOGDEBUG("loadConfigByHost|" << appServerName << "|" << host << "|" << filename << std::endl);
    auto v = TC_Common::sepstr<string>(appServerName, ".");
    try
    {
        if (v.size() == 1)
        {
            K8SInterface::loadConfig(v[0], "", filename, host, config);
        }
        else if (v.size() == 2)
        {
            K8SInterface::loadConfig(v[0], v[1], filename, host, config);
        }
        else
        {
            TLOGERROR("bad AppSeverName format|" << appServerName << endl);
            return -1;
        }
    }
    catch (const std::exception& e)
    {
        TLOGERROR(e.what() << endl);
        return -1;
    }
    return 0;
}

int
ConfigImp::loadConfig(const std::string& app, const std::string& server, const std::string& fileName, string& result,
        CurrentPtr current)
{
    TLOGDEBUG("loadConfig|" << app << "." << server << "|" << current->getIp() << "|" << fileName << std::endl);
    try
    {
        K8SInterface::loadConfig(app, server, fileName, current->getIp(), result);
    }
    catch (const std::exception& e)
    {
        TLOGERROR(e.what() << endl);
        return -1;
    }
    return 0;
}

int ConfigImp::checkConfig(const std::string& appServerName, const std::string& fileName, const std::string& host,
        string& result, CurrentPtr current)
{
    TLOGDEBUG("checkConfig|" << appServerName << "." << fileName << "|" << host << endl);
    if (loadConfigByHost(appServerName, fileName, host, result, current) != 0)
    {
        return -1;
    }
    try
    {
        TC_Config conf{};
        conf.parseString(result);
    }
    catch (const std::exception& ex)
    {
        result = ex.what();
        return -1;
    }
    return 0;
}

int ConfigImp::ListConfigByInfo(const ConfigInfo& configInfo, vector<string>& vf, CurrentPtr current)
{
    TLOGDEBUG("ListConfigByInfo|" << configInfo.appname << "." << configInfo.servername << "|" << configInfo.host
                                  << endl);
    try
    {
        if (configInfo.bAppOnly)
        {
            K8SInterface::listConfig(configInfo.appname, "", configInfo.host, vf);
        }
        else
        {
            K8SInterface::listConfig(configInfo.appname, configInfo.servername, configInfo.host, vf);
        }
    }
    catch (const std::exception& e)
    {
        TLOGERROR(e.what() << endl);
        return -1;
    }
    return 0;
}

int ConfigImp::loadConfigByInfo(const ConfigInfo& configInfo, string& config, CurrentPtr current)
{
    TLOGDEBUG("loadConfigByInfo|" << configInfo.appname << "|" << configInfo.servername << "|" << configInfo.filename
                                  << endl);
    try
    {
        if (configInfo.bAppOnly)
        {
            K8SInterface::loadConfig(configInfo.appname, "", configInfo.filename, configInfo.host, config);
        }
        else
        {
            K8SInterface::loadConfig(configInfo.appname, configInfo.servername, configInfo.filename, configInfo.host,
                    config);
        }
    }
    catch (const std::exception& e)
    {
        TLOGERROR(e.what() << endl);
        return -1;
    }
    return 0;
}

int ConfigImp::ListAllConfigByInfo(const GetConfigListInfo& configInfo, vector<std::string>& vf, CurrentPtr current)
{
    TLOGDEBUG("ListAllConfigByInfo|" << configInfo.appname << "." << configInfo.servername << "|" << configInfo.host
                                     << endl);
    try
    {
        if (configInfo.bAppOnly)
        {
            K8SInterface::listConfig(configInfo.appname, "", configInfo.host, vf);
        }
        K8SInterface::listConfig(configInfo.appname, configInfo.servername, configInfo.host, vf);
    }
    catch (const std::exception& e)
    {
        TLOGERROR(e.what() << endl);
        return -1;
    }
    return 0;
}

int ConfigImp::checkConfigByInfo(const ConfigInfo& configInfo, string& result, CurrentPtr current)
{
    TLOGDEBUG("checkConfigByInfo|" << configInfo.appname << "." << configInfo.servername << "|" << configInfo.host
                                   << endl);
    try
    {
        if (configInfo.bAppOnly)
        {
            K8SInterface::loadConfig(configInfo.appname, "", configInfo.filename, configInfo.host, result);
        }
        else
        {
            K8SInterface::loadConfig(configInfo.appname, configInfo.servername, configInfo.filename, configInfo.host,
                    result);
        }
        TC_Config conf{};
        conf.parseString(result);
    }
    catch (const std::exception& e)
    {
        result = e.what();
        return -1;
    }
    return 0;
}
