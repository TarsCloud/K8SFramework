#include "NodeServer.h"
#include "NodeImp.h"
#include "ServerImp.h"
#include "ServerObject.h"
#include "ProxyManger.h"
#include "Container.h"

void NodeServer::initialize()
{
    try
    {
        if (container::serverLauncherType == SERVER_FOREGROUND_LAUNCH)
        {
            if (target == TARSNODE_CONFIG_TARGET)
            {
                if (outfile.empty())
                {
                    std::cout << "should set outfile parameter when target is " << TARSNODE_CONFIG_TARGET << std::endl;
                    exit(-1);
                }

                if (ServerObject::generateTemplateConf() != 0)
                {
                    std::cout << "generate server template config file error" << std::endl;
                    exit(-1);
                }

                auto setting = ServerObject::generateLauncherSetting();
                auto&& launcherArgv = setting.argv_;
                std::fstream fs(outfile, std::fstream::out | std::fstream::trunc);
                if (fs.good())
                {
                    if (launcherArgv.size() > 1)
                    {
                        for (size_t i = 1; i < launcherArgv.size(); ++i)
                        {
                            fs << launcherArgv[i] << " ";
                        }
                        if (fs.fail())
                        {
                            std::cout << "write to outfile error" << std::endl;
                            exit(-1);
                        }
                    }
                }
                if (fs.fail())
                {
                    std::cout << "create or open outfile error" << std::endl;
                    exit(-1);
                }
                fs.close();
                exit(0);
            }

            if (target == TARSNODE_DAEMON_TARGET)
            {
                addServant<ServerImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".ServerObj");
                ServerObject::startForegroundPatrol();
                return;
            }
        }

        assert(container::serverLauncherType == SERVER_BACKGROUND_LAUNCH);
        addServant<NodeImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".NodeObj");
        addServant<ServerImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".ServerObj");
        ServerObject::startBackgroundPatrol();
        return;
    }
    catch (const TC_Exception& e)
    {
        TLOGERROR("NodeServer initialize exception: " << e.what() << std::endl);
        std::cout << "NodeServer initialize exception: " << e.what() << std::endl;
        exit(-1);
    }
}

void NodeServer::destroyApp()
{
    std::cout << "NodeServer::destroyApp ok" << std::endl;
}
