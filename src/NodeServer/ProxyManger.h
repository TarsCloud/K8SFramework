
#pragma once

#include <servant/QueryF.h>
#include <servant/NotifyF.h>

class ProxyManger {
public:
    static ProxyManger &instance() {
        static ProxyManger manger;
        return manger;
    }

    static AdminFPrx createAdminProxy(const std::string &sAdminProxyName) {
        AdminFPrx pAdminPrx = Application::getCommunicator()->stringToProxy<AdminFPrx>(sAdminProxyName);
        return pAdminPrx;
    }

    RegistryPrx getRegistryProxy() {
        RegistryPrx pRegistryPrx;
        Application::getCommunicator()->stringToProxy(_sRegistryProxyName, pRegistryPrx);
        return pRegistryPrx;
    }

    QueryFPrx getQueryProxy() {
        QueryFPrx pQueryPrx;
        Application::getCommunicator()->stringToProxy(_sQueryProxyName, pQueryPrx);
        return pQueryPrx;
    }

    NotifyPrx getNotifyProxy() {
        NotifyPrx pNotifyPrx;
        Application::getCommunicator()->stringToProxy(_sNotifyProxyName, pNotifyPrx);
        return pNotifyPrx;
    }

    inline void setRegistryObjName(const string &sRegistryProxyName) {
        _sRegistryProxyName = sRegistryProxyName;
    }

    inline void setQueryObjName(const string &sQueryObjProxyName) {
        _sQueryProxyName = sQueryObjProxyName;
    }

    inline void setNotifyObjName(const string &sNotifyObjProxyName) {
        _sNotifyProxyName = sNotifyObjProxyName;
    }

private:
    std::mutex _mutex;
    string _sRegistryProxyName;
    string _sQueryProxyName;
    string _sNotifyProxyName;

    ProxyManger() = default;
};
