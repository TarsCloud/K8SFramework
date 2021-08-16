
#include "LogServer.h"
#include <iostream>

using namespace tars;

LogServer g_app;

int main(int argc, char *argv[])
{
    try
    {
        g_app.main(argc, argv);
        g_app.waitForShutdown();
    }
    catch(exception &ex)
    {
        cout << ex.what() << endl;
    }

    return 0;
}


