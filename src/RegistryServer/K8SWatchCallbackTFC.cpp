
#include "K8SWatchCallback.h"
#include "Storage.h"
#include <iostream>

static std::shared_ptr<UPChain> buildUPChain(const boost::json::value& value)
{
    try
    {
        auto upc = boost::json::value_to<UPChain>(value);
        return std::make_shared<UPChain>(upc);
    }
    catch (const std::exception& e)
    {
        return nullptr;
    }
}

void Storage::updateUpChain(const shared_ptr<UPChain>& upChain)
{
    upChainMutex_.writeLock();
    upChain_ = upChain;
    upChainMutex_.unWriteLock();
}

void Storage::getUnChain(const function<void(std::shared_ptr<UPChain>&)>& f)
{
    upChainMutex_.readLock();
    f(upChain_);
    upChainMutex_.unReadLock();
}

void K8SWatchCallback::onTFCAdded(const boost::json::value& value, K8SWatchEventDrive driver)
{
    constexpr char ExpectedName[] = "tars-framework";
    auto upc = buildUPChain(value);
    if (upc != nullptr && upc->resourceName == ExpectedName)
    {
        Storage::instance().updateUpChain(upc);
    }
}

void K8SWatchCallback::onTFCModified(const boost::json::value& value)
{
    onTFCAdded(value, K8SWatchEventDrive::Watch);
}

void K8SWatchCallback::onTFCDeleted(const boost::json::value& value)
{
    constexpr char ExpectedName[] = "tars-framework";
    auto upc = buildUPChain(value);
    if (upc != nullptr && upc->resourceName == ExpectedName)
    {
        Storage::instance().updateUpChain(nullptr);
    }
}
