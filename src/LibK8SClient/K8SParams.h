#pragma once

#include "util/tc_file.h"
#include "asio/ssl/context.hpp"

using namespace tars;

struct K8SParams {
private:
    K8SParams() : _sslContext(asio::ssl::context::sslv23_client) {}

public:
    static K8SParams &instance() {
        static K8SParams runtimeParams;
        return runtimeParams;
    }

    void init() {

        constexpr char kubernetesServiceHostEnv[] = "KUBERNETES_SERVICE_HOST";
        constexpr char kubernetesServicePortEnv[] = "KUBERNETES_SERVICE_PORT";
        constexpr char tokenFile[] = "/var/run/secrets/kubernetes.io/serviceaccount/token";
        constexpr char caFile[] = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt";
        constexpr char namespaceFile[] = "/var/run/secrets/kubernetes.io/serviceaccount/namespace";

        std::string cert = TC_File::load2str(caFile);

        _sslContext.add_certificate_authority(asio::buffer(cert));

        _apiServerIp = getenv(kubernetesServiceHostEnv);

        const char *port = getenv(kubernetesServicePortEnv);

        _apiServerPort = std::stoi(port);

        _token = TC_File::load2str(tokenFile);

        _namespace = TC_File::load2str(namespaceFile);
    }

    const std::string &bindToken() {
        return _token;
    }

    const std::string &apiServerHost() {
        return _apiServerIp;
    }

    int apiServerPort() const {
        return _apiServerPort;
    }

    const std::string &bindNamespace() {
        return _namespace;
    }

    asio::ssl::context &sslContext() {
        return _sslContext;
    }

private:
    asio::ssl::context _sslContext;
    std::string _token{};
    std::string _apiServerIp{};
    std::string _namespace{};
    int _apiServerPort{};
};
