
#include "NodeServer.h"
#include <iostream>

using namespace tars;

int main(int argc, char *argv[]) {
    try {
        NodeServer app;
        app.main(argc, argv);
        app.waitForShutdown();
    }
    catch (exception &ex) {
        cerr << ex.what() << endl;
    }
    return 0;
}
