

#include "K8SWatchCallback.h"
#include "Storage.h"

static std::shared_ptr<TEndpoint> buildTE(const boost::json::value& value)
{
    try
    {
        auto te = boost::json::value_to<TEndpoint>(value);
        return std::make_shared<TEndpoint>(te);
    }
    catch (const std::exception& e)
    {
        return nullptr;
    }
}

void Storage::preListTEndpoint()
{
    cacheTEs_.clear();
}

void Storage::postListTEndpoint()
{
    {
        teMutex_.writeLock();
        std::swap(tes_, cacheTEs_);
        teMutex_.unWriteLock();
    }
    cacheTEs_.clear();
}

void Storage::updateTEndpoint(const string& name, const shared_ptr<TEndpoint>& t, K8SWatchEventDrive driver)
{
    if (driver == K8SWatchEventDrive::List)
    {
        cacheTEs_[name] = t;
        return;
    }
    teMutex_.writeLock();
    tes_[name] = t;
    teMutex_.unWriteLock();
}

void Storage::deleteTEndpoint(const string& name)
{
    teMutex_.writeLock();
    tes_.erase(name);
    teMutex_.unWriteLock();
}

void Storage::getTEndpoints(const function<void(const std::map<std::string, std::shared_ptr<TEndpoint>>&)>& f)
{
    teMutex_.readLock();
    f(tes_);
    teMutex_.unReadLock();
}

void K8SWatchCallback::preTEList()
{
    Storage::instance().preListTEndpoint();
}

void K8SWatchCallback::postTEList()
{
    Storage::instance().postListTEndpoint();
}

void K8SWatchCallback::onTEAdded(const boost::json::value& value, K8SWatchEventDrive driver)
{
    auto te = buildTE(value);
    if (te != nullptr)
    {
        Storage::instance().updateTEndpoint(te->resourceName, te, driver);
    }
}

void K8SWatchCallback::onTEModified(const boost::json::value& value)
{
    return onTEAdded(value, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onTEDeleted(const boost::json::value& value)
{
    auto te = buildTE(value);
    if (te != nullptr)
    {
        Storage::instance().deleteTEndpoint(te->resourceName);
    }
}
