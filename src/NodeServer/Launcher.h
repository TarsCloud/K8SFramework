
#pragma once

class Launcher {
private:
    struct EnvironmentEval : std::unary_function<string, string> {
        string
        operator()(const std::string &value) {
            string::size_type assignment = value.find('=');
            if (assignment == string::npos || assignment >= value.size() - 1) {
                return value;
            }
            string v = value.substr(assignment + 1);
            string::size_type beg = 0;
            string::size_type end;

            while ((beg = v.find('$', beg)) != string::npos && beg < v.size() - 1) {
                string variable;
                if (v[beg + 1] == '{') {
                    end = v.find('}');
                    if (end == string::npos) {
                        break;
                    }
                    variable = v.substr(beg + 2, end - beg - 2);
                } else {
                    end = beg + 1;
                    while ((isalnum(v[end]) || v[end] == '_') && end < v.size()) {
                        ++end;
                    }
                    variable = v.substr(beg + 1, end - beg - 1);
                    --end;
                }
                char *val = getenv(variable.c_str());
                string sVal = val ? string(val) : "";
                v.replace(beg, end - beg + 1, sVal);
                beg += sVal.size();
            }

            setenv((value.substr(0, assignment)).c_str(), v.c_str(), true);
            return value.substr(0, assignment) + "=" + v;
        }
    };

public:
    static pid_t activate(const std::string &sLauncherFile,
                          const string &sPwdPath,
                          const string &sRedirectFile,
                          const std::vector<string> &vLauncherArgv,
                          const vector<string> &vEnvs) {

        if (sLauncherFile.empty() || !TC_File::isFileExist(sLauncherFile)) {
            throw runtime_error("executable is empty or not exist");
        }

        pid_t pid = fork();
        if (pid == -1) {
            throw runtime_error("fork child process error");
        }

        if (pid == 0) {
            int maxFd = static_cast<int>(sysconf(_SC_OPEN_MAX));
            for (int fd = 3; fd < maxFd; ++fd) {
                close(fd);
            }

            if (!sRedirectFile.empty()) {

                const std::string redirectPath = TC_File::extractFilePath(sRedirectFile);

                if (!TC_File::makeDirRecursive(redirectPath)) {
                    cerr << "create dir [" << redirectPath << "] error" << endl;
                    exit(0);
                }

#if TARGET_PLATFORM_IOS
                if ((freopen(sRedirectFile.c_str(), "ab", stdout)) != nullptr && (freopen(sRedirectFile.c_str(), "ab", stderr)) != nullptr) {
#else
                if ((freopen64(sRedirectFile.c_str(), "ab", stdout)) != nullptr && (freopen64(sRedirectFile.c_str(), "ab", stderr)) != nullptr) {
#endif
                    cerr << "redirect stdout and stderr to " << sRedirectFile << endl;
                } else {
                    exit(0);
                }
            }

            if (!sPwdPath.empty() && (chdir(sPwdPath.c_str()) == -1)) {
                cerr << "cannot chdir to " << sPwdPath << "|errno=" << strerror(errno) << endl;
                exit(0);
            }

            for_each(vEnvs.begin(), vEnvs.end(), EnvironmentEval());

            vector<const char *> vArgv(vLauncherArgv.size() + 1);
            for (size_t i = 0; i < vLauncherArgv.size(); ++i) {
                vArgv[i] = vLauncherArgv[i].c_str();
            }

            char *const *argv = const_cast<char *const *>(&vArgv[0]);

            if (execvp(sLauncherFile.c_str(), argv) == -1) {
                cerr << "cannot execute " << sLauncherFile << "|errno=" << strerror(errno) << endl;
                exit(0);
            }
            exit(0);
        }
        return pid;
    }
};
