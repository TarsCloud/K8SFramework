
#include "K8SWatchCallback.h"
#include "Storage.h"

std::shared_ptr<TTemplate> buildTT(const boost::json::value& value)
{
    try
    {
        auto tt = boost::json::value_to<TTemplate>(value);
        return std::make_shared<TTemplate>(tt);
    }
    catch (const std::exception& e)
    {
        return nullptr;
    }
}

void Storage::preListTTemplate()
{
    cacheTTs_.clear();
}

void Storage::postListTTemplate()
{
    {
        ttMutex_.writeLock();
        std::swap(tts_, cacheTTs_);
        ttMutex_.unWriteLock();
    }
    cacheTTs_.clear();
}

void Storage::updateTTemplate(const string& name, const shared_ptr<TTemplate>& t, K8SWatchEventDrive driver)
{
    if (driver == K8SWatchEventDrive::List)
    {
        cacheTTs_[t->name] = t;
        return;
    }
    ttMutex_.writeLock();
    tts_[t->name] = t;
    ttMutex_.unWriteLock();
}

void Storage::deleteTTemplate(const string& name)
{
    ttMutex_.writeLock();
    tts_.erase(name);
    ttMutex_.unWriteLock();
}

void Storage::getTTemplates(const function<void(const std::map<std::string, std::shared_ptr<TTemplate>>&)>& f)
{
    ttMutex_.readLock();
    f(tts_);
    ttMutex_.unReadLock();
}

void K8SWatchCallback::preTTList()
{
    Storage::instance().preListTTemplate();
}

void K8SWatchCallback::postTTList()
{
    Storage::instance().postListTTemplate();
}

void K8SWatchCallback::onTTAdded(const boost::json::value& value, K8SWatchEventDrive driver)
{
    auto tt = buildTT(value);
    if (tt != nullptr)
    {
        Storage::instance().updateTTemplate(tt->name, tt, driver);
    }
}

void K8SWatchCallback::onTTModified(const boost::json::value& value)
{
    return onTTAdded(value, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onTTDeleted(const boost::json::value& value)
{
    auto tt = buildTT(value);
    if (tt != nullptr)
    {
        Storage::instance().deleteTTemplate(tt->name);
    }
}
