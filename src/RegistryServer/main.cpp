#include "RegistryServer.h"
#include <iostream>

int main(int argc, char* argv[])
{
    try
    {
        RegistryServer app;
        app.main(argc, argv);
        app.waitForShutdown();
    }
    catch (exception& ex)
    {
        cerr << ex.what() << endl;
    }
    return 0;
}
