#include "ConfigImp.h"
#include "ConfigServer.h"
#include "ConfigInfoInterface.h"
#include "servant/Application.h"
using namespace tars;

void ConfigImp::initialize() {
}

int ConfigImp::ListConfig(const string &app, const string &server, vector<string> &vf, CurrentPtr current) {
    TLOG_DEBUG(app << "." << server << endl);

    std::string errorInfo;
    auto loadConfigResult = ConfigInfoInterface::instance().listConfig(app, server, vf, errorInfo);

    if (loadConfigResult == ConfigInfoInterface::ConfigError) {
        TLOG_DEBUG("error: " << errorInfo << ", app:" << app << ", server:" << server << std::endl);
        return -1;
    } else if (loadConfigResult == ConfigInfoInterface::K8SError) {
        TARS_NOTIFY_ERROR("request k8s api-server error:" + errorInfo);
        TLOG_ERROR("request k8s api-server error : " << errorInfo << ", app:" << app << ", server:" << server << std::endl);
        return -1;
    }
    return 0;
}

int ConfigImp::loadConfigByHost(const std::string &appServerName, const std::string &fileName, const string &host,
                                string &result, CurrentPtr current) {
    auto v = TC_Common::sepstr<string>(appServerName, ".");

    std::string errorInfo;
    ConfigInfoInterface::GetConfigResult loadConfigResult;

    if (v.size() == 1) {
        loadConfigResult = ConfigInfoInterface::instance().loadConfig(v[0], "", fileName, host, result, errorInfo);
    } else if (v.size() == 2) {
        loadConfigResult = ConfigInfoInterface::instance().loadConfig(v[0], v[1], fileName, host, result, errorInfo);
    } else {
        result.append("bad request format : ").append(appServerName);
        TLOG_ERROR(result << ", fileName:" << fileName << ", host:" << host << endl);
        return -1;
    }

    if (loadConfigResult == ConfigInfoInterface::ConfigError) {
        TLOG_DEBUG("load config error: " << errorInfo << ", appServerName:" << appServerName << ", fileName:" << fileName << ", host:" << host << std::endl);
        std::swap(result, errorInfo);
        return -1;
    } else if (loadConfigResult == ConfigInfoInterface::K8SError) {
        TARS_NOTIFY_ERROR("request k8s api-server error:" + errorInfo);
        TLOG_ERROR("request k8s api-server error : " << errorInfo << ", appServerName:" << appServerName << ", fileName:" << fileName << ", host:" << host << std::endl);
        result = "internal error, please try again or contact administrator";
        return -1;
    }
    return 0;
}

int
ConfigImp::loadConfig(const std::string &app, const std::string &server, const std::string &fileName, string &result,
                      CurrentPtr current) {
    std::string sClientIP = current->getIp();

    std::string errorInfo;
    auto loadConfigResult = ConfigInfoInterface::instance().loadConfig(app, server, fileName, sClientIP, result,
                                                                       errorInfo);

    if (loadConfigResult == ConfigInfoInterface::ConfigError) {
        TLOG_DEBUG( "load config error: " << errorInfo << ", app:" << app << ", server:" << server << ", fileName:" << fileName << std::endl);
        std::swap(result, errorInfo);
        return -1;
    } else if (loadConfigResult == ConfigInfoInterface::K8SError) {
        TARS_NOTIFY_ERROR("request k8s api-server error:" + errorInfo);
        TLOG_ERROR("request k8s api-server error : " << errorInfo << ", app:" << app << ", server:" << server << ", fileName:" << fileName << std::endl);
        result = "internal error, please try again or contact  administrator";
        return -1;
    }
    return 0;
}

int ConfigImp::checkConfig(const std::string &appServerName, const std::string &fileName, const string &host,
                           string &result, CurrentPtr current) {

    std::string errorInfo;
    ConfigInfoInterface::GetConfigResult loadConfigResult;

    auto v = TC_Common::sepstr<string>(appServerName, ".");
    if (v.size() == 1) {
        loadConfigResult = ConfigInfoInterface::instance().loadConfig(v[0], "", fileName, host, result, errorInfo);
    } else if (v.size() == 2) {
        loadConfigResult = ConfigInfoInterface::instance().loadConfig(v[0], v[1], fileName, host, result, errorInfo);
    } else {
        result.append("bad request format : ").append(appServerName);
        TLOG_ERROR(result << ", fileName:" << fileName << ", host:" << host << endl);
        return -1;
    }

    if (loadConfigResult == ConfigInfoInterface::ConfigError) {
        TLOG_DEBUG( "load config error: " << errorInfo << ", appServerName:" << appServerName << ", fileName:" << fileName << ", host:" << host << std::endl);
        std::swap(result, errorInfo);
        return -1;
    } else if (loadConfigResult == ConfigInfoInterface::K8SError) {
        TARS_NOTIFY_ERROR("request k8s api-server error:" + errorInfo);
        TLOG_ERROR("request k8s api-server error : " << errorInfo << ", appServerName:" << appServerName << ", fileName:" << fileName << ", host:" << host << std::endl);
        result = "internal error, please try again or contact  administrator";
        return -1;
    }
    try {
        TC_Config conf;
        conf.parseString(result);
    }
    catch (exception &ex) {
        TLOG_ERROR("error:" << ex.what() << ", appServerName:" << appServerName << ", fileName:" << fileName << ", host:" << host << endl);
        result = ex.what();
        return -1;
    }
    return 0;
}

int ConfigImp::ListConfigByInfo(const ConfigInfo &configInfo, vector<string> &vf, CurrentPtr current) {
    TLOG_DEBUG(configInfo.appname << "|" << configInfo.servername << endl);

    std::string errorInfo;
    ConfigInfoInterface::GetConfigResult loadConfigResult;

    if (configInfo.bAppOnly) {
        loadConfigResult = ConfigInfoInterface::instance().listConfig(configInfo.appname, "", vf, errorInfo);
    } else {
        loadConfigResult = ConfigInfoInterface::instance().listConfig(configInfo.appname, configInfo.servername, vf,
                                                                      errorInfo);
    }

    if (loadConfigResult == ConfigInfoInterface::ConfigError) {
        TLOG_DEBUG( "list config error: " << errorInfo << ", appName:" << configInfo.appname << ", serverName:" << configInfo.servername << std::endl);
        return -1;
    } else if (loadConfigResult == ConfigInfoInterface::K8SError) {
        TARS_NOTIFY_ERROR("request k8s api-server error:" + errorInfo);
        TLOG_ERROR("request k8s api-server error : " << errorInfo << ", appName:" << configInfo.appname << ", serverName:" << configInfo.servername << std::endl);
        return -1;
    }
    return 0;
}

int ConfigImp::loadConfigByInfo(const ConfigInfo &configInfo, string &config, CurrentPtr current) {
    TLOG_DEBUG( configInfo.appname << "|" << configInfo.servername << "|"
                 << configInfo.filename << endl);

    std::string errorInfo;
    ConfigInfoInterface::GetConfigResult loadConfigResult;

    if (configInfo.bAppOnly) {
        loadConfigResult = ConfigInfoInterface::instance().loadConfig(configInfo.appname, "", configInfo.filename,
                                                                      configInfo.host, config, errorInfo);
    } else {
        loadConfigResult = ConfigInfoInterface::instance().loadConfig(configInfo.appname, configInfo.servername,
                                                                      configInfo.filename, configInfo.host, config,
                                                                      errorInfo);
    }

    if (loadConfigResult == ConfigInfoInterface::ConfigError) {
        std::swap(config, errorInfo);
        return -1;
    } else if (loadConfigResult == ConfigInfoInterface::K8SError) {
        TARS_NOTIFY_ERROR("request k8s api-server error:" + errorInfo);
        TLOG_ERROR("request k8s api-server error : " << errorInfo << ", appName:" << configInfo.appname << ", serverName:" << configInfo.servername << ", fileName:"
                 << configInfo.filename << std::endl);
        return -1;
    }
    return 0;
}

Int32 ConfigImp::ListAllConfigByInfo(const GetConfigListInfo &configInfo, vector<std::string> &vf, CurrentPtr current) {
    TLOG_DEBUG(configInfo.appname << "|" << configInfo.servername << endl);
    if (configInfo.bAppOnly) {
        return ListConfig(configInfo.appname, "", vf, current);
    }
    return ListConfig(configInfo.appname, configInfo.servername, vf, current);
}

int ConfigImp::checkConfigByInfo(const ConfigInfo &configInfo, string &result, CurrentPtr current) {

    std::string errorInfo;

    auto loadConfigResult = ConfigInfoInterface::instance().loadConfig(configInfo.appname, configInfo.servername,
                                                                       configInfo.filename,
                                                                       configInfo.host, result, errorInfo);
    if (loadConfigResult == ConfigInfoInterface::ConfigError) {
        TLOG_DEBUG( "load config error: " << errorInfo << ", appName:" << configInfo.appname << ", serverName:" << configInfo.servername << ", fileName:"
                                                                       << configInfo.filename << ", host:" << configInfo.host << std::endl);
        std::swap(result, errorInfo);
        return -1;
    } else if (loadConfigResult == ConfigInfoInterface::K8SError) {
        TARS_NOTIFY_ERROR("request k8s api-server error:" + errorInfo);
        TLOG_ERROR("request k8s api-server error : " << errorInfo << ", appName:" << configInfo.appname << ", serverName:" << configInfo.servername << ", fileName:"
                                                                       << configInfo.filename << ", host:" << configInfo.host << std::endl);
        result = "internal error, please try again or contact  administrator";
        return -1;
    }
    try {
        TC_Config conf;
        conf.parseString(result);
    }
    catch (exception &ex) {
        TLOG_ERROR("error:" << ex.what() << ", appName:" << configInfo.appname << ", serverName:" << configInfo.servername << ", fileName:"
                                                                       << configInfo.filename << ", host:" << configInfo.host<< endl);

        result = ex.what();
        return -1;
    }
    return 0;
}
