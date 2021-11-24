
#pragma once

#include <string>
#include "Registry.h"

using namespace tars;

class RegistryImp : public Registry
{
public:
	RegistryImp() = default;;

	void initialize() override;

	void destroy() override
	{
	};

	Int32
	getServerDescriptor(const std::string& serverApp, const std::string& serverName, ServerDescriptor& serverDescriptor, CurrentPtr current) override;

	void updateServerState(const std::string& podName, const std::string& settingState, const std::string& presentState, CurrentPtr current) override;
};
