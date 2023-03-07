#include "K8SParams.h"
#include <fstream>
#include <thread>

constexpr char KubernetesServiceHostEnv[] = "KUBERNETES_SERVICE_HOST";
constexpr char KubernetesServicePortEnv[] = "KUBERNETES_SERVICE_PORT";
constexpr char TokenFile[] = "/var/run/secrets/kubernetes.io/serviceaccount/token";
constexpr char CaFile[] = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt";
constexpr char NamespaceFile[] = "/var/run/secrets/kubernetes.io/serviceaccount/namespace";

static std::string loadFile(const std::string& file, size_t MAX_SIZE = 8192)
{
    auto fs = ::fopen(file.c_str(), "r");
    if (fs == nullptr)
    {return "";}
    std::unique_ptr<char[]> p(new char[MAX_SIZE]);
    auto read_size = ::fread(p.get(), 1, MAX_SIZE, fs);
    ::fclose(fs);
    return read_size > 0 ? std::string{ p.get(), read_size } : "";
}

struct K8SParamsIMP
{
public:
    static K8SParamsIMP& instance()
    {
        static K8SParamsIMP imp{};
        return imp;
    };

    const std::string& apiServerHost()
    {
        return _apiServerHost;
    }

    int apiServerPort() const
    {
        return _apiServerPort;
    }

    const std::string& Namespace()
    {
        return _namespace;
    }

    boost::asio::ssl::context& sslContext()
    {
        return _sslContext;
    };

private:
    K8SParamsIMP()
            :_sslContext(boost::asio::ssl::context::sslv23_client)
    {
        auto pHost = getenv(KubernetesServiceHostEnv);
        if (pHost == nullptr)
        {
            throw std::runtime_error("should");
        }
        _apiServerHost = std::string{ pHost };

        const char* pPort = getenv(KubernetesServicePortEnv);
        if (pPort == nullptr)
        {
            throw std::runtime_error("should");
        }
        _apiServerPort = std::stoi(pPort);

        _namespace = loadFile(NamespaceFile);
        _sslContext.add_certificate_authority(boost::asio::buffer(loadFile(CaFile)));
    }

private:
    boost::asio::ssl::context _sslContext;
    std::string _apiServerHost{};
    std::string _namespace{};
    int _apiServerPort{};
};

const std::string& K8SParams::APIServerHost()
{
    return K8SParamsIMP::instance().apiServerHost();
}

int K8SParams::APIServerPort()
{
    return K8SParamsIMP::instance().apiServerPort();
}

const std::string& K8SParams::Namespace()
{
    return K8SParamsIMP::instance().Namespace();
}

boost::asio::ssl::context& K8SParams::SSLContext()
{
    return K8SParamsIMP::instance().sslContext();
}

const std::string K8SParams::ClientToken()
{
    constexpr int MAX_TRIES = 3;
    for (auto i = 0; i < MAX_TRIES; ++i)
    {
        auto token = loadFile(TokenFile);
        if (!token.empty())
        {
            return token;
        }
        std::this_thread::sleep_for(std::chrono::microseconds(10));
    }
    return "";
}
