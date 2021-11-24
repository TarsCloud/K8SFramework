
#pragma once

#include "servant/Application.h"
#include <vector>
#include <tuple>
#include <string>

enum class ESClientRequestMethod
{
	Put,
	Post,
	Get,
};

class ESClient
{
public:
	static ESClient& instance()
	{
		static ESClient esClient;
		return esClient;
	}

	void setAddresses(const std::vector<std::tuple<std::string, int>>& addresses, const std::string& protocol = "http");

	int doRequest(ESClientRequestMethod method, const std::string& url, const std::string& body, std::string& response);

private:
	ESClient() = default;

private:
	ServantPrx _esPrx{};
};