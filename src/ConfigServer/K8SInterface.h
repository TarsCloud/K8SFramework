#pragma once

#include <string>
#include <vector>

class K8SInterface
{
public:
	static void
	listConfig(const std::string& app, const std::string& server, const std::string& host, std::vector<std::string>& vf)
	noexcept(false);

	static void
	loadConfig(const std::string& app, const std::string& server, const std::string& fileName, const std::string& host, std::string& result)
	noexcept(false);
};
