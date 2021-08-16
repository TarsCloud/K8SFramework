#include "RegistryServer.h"
#include <iostream>

using namespace tars;

int main(int argc, char *argv[]) {
    try {
        RegistryServer g_app;
        g_app.main(argc, argv);
        g_app.waitForShutdown();
    }
    catch (exception &ex) {
        cerr << ex.what() << endl;
    }
    return 0;
}
