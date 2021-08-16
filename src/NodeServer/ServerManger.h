
#pragma once

#include "ServerObject.h"
#include <vector>
#include <mutex>

class ServerManger {
public:
    static ServerManger &instance() {
        static ServerManger manger;
        return manger;
    }

    const ServerObjectPtr &getServer(const std::string &sServerApp, const std::string &sServerName) {
        std::lock_guard<std::mutex> lockGuard(_mutex);
        if (sServerApp.empty() || sServerName.empty() ||
            _pServerObjectPtr == nullptr ||
            sServerApp != _pServerObjectPtr->getApplication() ||
            sServerName != _pServerObjectPtr->getServerName()) {
            return _nullServerObjPtr;
        }
        return _pServerObjectPtr;
    }

    void putServer(const std::shared_ptr<ServerObject> &ptr) {
        std::lock_guard<std::mutex> lockGuard(_mutex);
        _pServerObjectPtr = ptr;
    }

private:
    ServerManger() = default;

private:
    std::mutex _mutex;
    ServerObjectPtr _nullServerObjPtr{nullptr};
    ServerObjectPtr _pServerObjectPtr{nullptr};
};
