#pragma once

#include <servant/RemoteLogger.h>

struct LauncherSetting
{
    std::string file_;
    string workDir_;
    string redirect_;
    std::vector<string> argv_;
    std::vector<std::string> envs_;
};

class Launcher
{
private:
    struct EnvironmentEval : std::unary_function<string, string>
    {
        string
        operator()(const std::string& value)
        {
            string::size_type assignment = value.find('=');
            if (assignment == string::npos || assignment >= value.size() - 1)
            {
                return value;
            }
            string v = value.substr(assignment + 1);
            string::size_type beg = 0;
            string::size_type end;

            while ((beg = v.find('$', beg)) != string::npos && beg < v.size() - 1)
            {
                string variable;
                if (v[beg + 1] == '{')
                {
                    end = v.find('}');
                    if (end == string::npos)
                    {
                        break;
                    }
                    variable = v.substr(beg + 2, end - beg - 2);
                }
                else
                {
                    end = beg + 1;
                    while ((isalnum(v[end]) || v[end] == '_') && end < v.size())
                    {
                        ++end;
                    }
                    variable = v.substr(beg + 1, end - beg - 1);
                    --end;
                }
                char* val = getenv(variable.c_str());
                string sVal = val ? string(val) : "";
                v.replace(beg, end - beg + 1, sVal);
                beg += sVal.size();
            }

            setenv((value.substr(0, assignment)).c_str(), v.c_str(), true);
            return value.substr(0, assignment) + "=" + v;
        }
    };

public:
    static pid_t activate(const LauncherSetting& setting)
    {
        if (setting.file_.empty() || !TC_File::isFileExist(setting.file_))
        {
            TLOGERROR("executable is empty or not exist" << std::endl);
            std::cout << "executable is empty or not exist" << std::endl;
            return -1;
        }

        pid_t pid = ::fork();

        if (pid == -1)
        {
            TLOGERROR("::fork() failed" << std::endl);
            std::cout << "::fork() failed" << std::endl;
            return -1;
        }

        if (pid == 0)
        {
            /* Fixme
                In the high kernel version (>=5.3) system, we can use close_range
                But in order to be compatible with low-version kernel systems,
                We use a fake maximum value, which is enough for tarsnode
            */
            //int maxFd = static_cast<int>(sysconf(_SC_OPEN_MAX));
            constexpr int FakeButWorkMaxFd = 10000;
            constexpr int maxFd = FakeButWorkMaxFd;
            for (int fd = 3; fd < maxFd; ++fd)
            {
                close(fd);
            }

            if (!setting.redirect_.empty())
            {
                const std::string redirectPath = TC_File::extractFilePath(setting.redirect_);
                if (!TC_File::makeDirRecursive(redirectPath))
                {
                    TLOGERROR("create dir \"" << redirectPath << "\" error" << std::endl);
                    std::cout << "create dir \"" << redirectPath << "\" error" << std::endl;
                    exit(-1);
                }

                if ((freopen64(setting.redirect_.c_str(), "ab", stdout)) == nullptr ||
                    (freopen64(setting.redirect_.c_str(), "ab", stderr)) == nullptr)
                {
                    TLOGERROR("redirect stdout and stderr to " << setting.redirect_ << "error" << std::endl);
                    std::cout << "redirect stdout and stderr to " << setting.redirect_ << "error" << std::endl;
                    exit(-1);
                }
            }

            if (!setting.workDir_.empty() && chdir(setting.workDir_.c_str()) == -1)
            {
                TLOGERROR("chdir to \"" << setting.workDir_ << "\" error" << std::endl);
                std::cout << "chdir to \"" << setting.workDir_ << "\" error" << endl;
                exit(-1);
            }

            for_each(setting.envs_.begin(), setting.envs_.end(), EnvironmentEval());

            vector<const char*> vArgv(setting.argv_.size() + 1);
            for (size_t i = 0; i < setting.argv_.size(); ++i)
            {
                vArgv[i] = setting.argv_[i].c_str();
            }

            char* const* argv = const_cast<char* const*>(&vArgv[0]);
            execvp(setting.file_.c_str(), argv);

            TLOGERROR("cannot execute " << setting.file_ << "|errno=" << strerror(errno) << std::endl);
            std::cout << "cannot execute " << setting.file_ << "|errno=" << strerror(errno) << std::endl;
            exit(-1);
        }

        return pid;
    }
};
