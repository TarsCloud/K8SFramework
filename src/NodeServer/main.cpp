
#include "Container.h"
#include "NodeServer.h"
#include "ServerObject.h"
#include "Fixed.h"
#include <iostream>

using namespace tars;

int main(int argc, char* argv[])
{
    if (!container::loadValues())
    {
        TLOGERROR("loadValues error" << std::endl);
        std::cout << "loadValues error";
        exit(-1);
    }

    std::string target{};
    std::string outfile{};

    if (container::serverLauncherType == SERVER_FOREGROUND_LAUNCH)
    {
        TC_Option option{};
        option.decode(argc, argv);

        target = option.getValue("target");
        outfile = option.getValue("outfile");

        if (target.empty())
        {
            std::cout << "should set target parameter when ServerLauncherType is " << SERVER_FOREGROUND_LAUNCH
                      << std::endl;
            exit(-1);
        }

        if (target != TARSNODE_CONFIG_TARGET && target != TARSNODE_DAEMON_TARGET)
        {
            std::cout << "unknown target value|" << target << std::endl;
            exit(-1);
        }
    }

    try
    {
        NodeServer app;
        app.target = target;
        app.outfile = outfile;
        app.main(argc, argv);
        app.waitForShutdown();
    }
    catch (exception& ex)
    {
        cerr << ex.what() << endl;
    }
    return 0;
}
