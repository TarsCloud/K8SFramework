
#pragma once

#include "Registry.h"
#include <string>

class RegistryImp : public Registry
{
public:
    RegistryImp() = default;

    void initialize() override;

    void destroy() override;

    int32_t getServerDescriptor(const std::string& endpoints, const std::string& serverName,
            tars::ServerDescriptor& serverDescriptor,
            CurrentPtr current) override;

    void updateServerState(const std::string& podName, const std::string& settingState, const std::string& presentState,
            CurrentPtr current) override;
};
