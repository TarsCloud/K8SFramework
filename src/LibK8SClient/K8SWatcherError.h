#include <system_error>

enum class K8SWatcherError
{
    Success = 0,
    BadParams,
    UnexpectedResponse,
};

class K8SWatcherErrorCategory : public std::error_category
{
public:
    static K8SWatcherErrorCategory const& instance()
    {
        static K8SWatcherErrorCategory instance;
        return instance;
    }

    char const* name() const

    noexcept override
            {
                    return "K8SWatcherError";
            }

    std::string message(int code) const override
    {
        auto error = static_cast<K8SWatcherError>(code);
        switch (error)
        {
        case K8SWatcherError::Success:
            return "Success";
        case K8SWatcherError::BadParams:
            return "Bas Params";
        case K8SWatcherError::UnexpectedResponse:
            return "Unexpected Response";
        }
        return { "Unknown Error" };
    }

    bool equivalent(std::error_code const& code, int condition) const noexcept override
    {
        if (code.category() != K8SWatcherErrorCategory::instance())
        {
            return false;
        }
        return true;
    }
};

namespace std
{
    template<>
    struct is_error_code_enum<K8SWatcherError> : true_type
    {
    };
}

inline std::error_code make_error_code(K8SWatcherError code)
{
    return { static_cast<int>(code), K8SWatcherErrorCategory::instance() };
}
