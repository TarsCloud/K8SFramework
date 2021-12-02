
#pragma once

#include "ESClient.h"
#include "util/tc_config.h"
#include "util/tc_timer.h"

struct ESRequestContext
{
	ssize_t times{};
	std::string uri{};
	std::string body{};
};

class ESHelper
{
public:
	static void post2ESWithRetry(TC_Timer* timer, const std::shared_ptr<ESRequestContext>& context, ssize_t maxRetry = 10)
	{
		assert(timer != nullptr);
		if (context->body.empty())
		{
			return;
		}
		std::string response{};
		ssize_t nexTimer{};
		int res = ESClient::instance().doRequest(ESClientRequestMethod::Post, context->uri, context->body, response);
		if (res != 200)
		{
			if (context->times <= maxRetry)
			{
				TLOGERROR("do elk request error: " << response << ", this is " << context->times << "th" << " retry" << endl);
				nexTimer = context->times * context->times + 1u;
			}
			else
			{
				TLOGERROR("do elk request error: " << response << ", this is " << context->times << "th" << " retry, request will discard" << endl);
				return;
			}
		}

		if (nexTimer != 0)
		{
			++context->times;
			timer->postDelayed(nexTimer * 1000, [timer, context]()
			{
				post2ESWithRetry(timer, context);
			});
		}
	}

	static void setAddressByTConfig(const TC_Config& config)
	{
		vector<string> nodes = config.getDomainKey("/tars/elk/nodes");
		if (nodes.empty())
		{
			TLOGERROR("empty elk nodes" << std::endl);
			std::cout << "empty elk nodes" << std::endl;
			throw std::runtime_error("empty elk nodes");
		}
		std::vector<std::tuple<std::string, int>> esNodes;
		for (auto& item: nodes)
		{
			vector<string> v = TC_Common::sepstr<string>(item, ":", true);
			if (v.size() < 2)
			{
				TLOGERROR("wrong elk node: " << item << endl);
				continue;
			}
			esNodes.emplace_back(v[0], std::stoi(v[1]));
		}
		if (esNodes.empty())
		{
			TLOGERROR("empty elk nodes" << std::endl);
			std::cout << "empty elk nodes" << std::endl;
			throw std::runtime_error("empty elk nodes");
		}

		string proto = config.get("/tars/elk<protocol>", "http");
		ESClient::instance().setAddresses(esNodes, proto);
	}

	static void createESPolicy(const std::string& name, const std::string& age)
	{
		constexpr char Template[] = R"(
{
  "policy": {
    "phases": {
      "delete": {
        "min_age": "{_AGE_}",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
)";

		auto url = std::string("/_ilm/policy/").append(name);
		std::string agePolicyContent = Template;
		auto body = tars::TC_Common::replace(agePolicyContent, "{_AGE_}", age);
		std::string response{};
		int res = ESClient::instance().doRequest(ESClientRequestMethod::Put, url, body, response);
		if (res != 200)
		{
			auto& message = std::string("create elk policy error, ").append(response);
			TLOGERROR(message << std::endl);
			throw std::runtime_error(message);
		}
	}

	static void createESIndexTemplate(const std::string& name, const std::string& pattern, const std::string& policy)
	{
		constexpr char Template[] = R"(
{
    "index_patterns":["{_PATTERN_}"],
    "template":{
        "settings":{
            "index.lifecycle.name":"{_POLICY_}"
        }
    }
}
)";

		auto url = std::string("/_index_template/").append(name);
		std::string TemplateContent = Template;
		auto body = tars::TC_Common::replace(TemplateContent, {{ "{_PATTERN_}", pattern },
															   { "{_POLICY_}",  policy }});
		std::string response{};
		int res = ESClient::instance().doRequest(ESClientRequestMethod::Put, url, body, response);
		if (res != 200)
		{
			auto message = std::string("create elk index template error, ").append(response);
			TLOGERROR(message << std::endl);
			throw std::runtime_error(message);
		}
	}

	static void createESDataStreamTemplate(const std::string& name, const std::string& pattern, const std::string& policy)
	{
		constexpr char Template[] = R"(
{
    "index_patterns":["{_PATTERN_}"],
    "data_stream":{},
    "template":{
        "settings":{
            "index.lifecycle.name":"{_POLICY_}"
        }
    }
}
)";

		auto url = std::string("/_index_template/").append(name);
		std::string TemplateContent = Template;
		auto body = tars::TC_Common::replace(TemplateContent, {{ "{_PATTERN_}", pattern },
															   { "{_POLICY_}",  policy }});
		std::string response{};
		int res = ESClient::instance().doRequest(ESClientRequestMethod::Put, url, body, response);
		if (res != 200)
		{
			auto message = std::string("create elk index template error, ").append(response);
			TLOGERROR(message << std::endl);
			throw std::runtime_error(message);
		}
	}
};
