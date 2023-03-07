
#pragma once

#include "servant/QueryF.h"
#include "servant/NotifyF.h"
#include "servant/AdminF.h"
#include "RegistryServer/Registry.h"
#include "servant/Application.h"
#include "Fixed.h"

class ProxyManger
{
public:
    static ProxyManger& instance()
    {
        static ProxyManger manger;
        return manger;
    }

    ~ProxyManger() = default;

    AdminFPrx getAdminProxy()
    {
        AdminFPrx pAdminPrx;
        Application::getCommunicator()->stringToProxy(FIXED_LOCAL_PROXY_NAME, pAdminPrx);
        return pAdminPrx;
    }

    RegistryPrx getRegistryProxy()
    {
        RegistryPrx pRegistryPrx;
        Application::getCommunicator()->stringToProxy(FIXED_REGISTRY_PROXY_NAME, pRegistryPrx);
        return pRegistryPrx;
    }

    QueryFPrx getQueryProxy()
    {
        QueryFPrx pQueryPrx;
        Application::getCommunicator()->stringToProxy(FIXED_QUERY_PROXY_NAME, pQueryPrx);
        return pQueryPrx;
    }

    NotifyPrx getNotifyProxy()
    {
        NotifyPrx pNotifyPrx;
        Application::getCommunicator()->stringToProxy(FIXED_NOTIFY_PROXY_NAME, pNotifyPrx);
        return pNotifyPrx;
    }

    ProxyManger() = default;
};
