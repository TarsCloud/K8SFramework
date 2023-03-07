#pragma once

#include <string>
#include <boost/asio/ssl/context.hpp>

struct K8SParams
{
    static const std::string ClientToken();

    static const std::string& APIServerHost();

    static int APIServerPort();

    static const std::string& Namespace();

    static boost::asio::ssl::context& SSLContext();
};
