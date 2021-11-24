
#include "QueryServer.h"
#include <iostream>

int main(int argc, char* argv[])
{
	try
	{
		QueryServer app{};
		app.main(argc, argv);
		RemoteTimeLogger::getInstance()->enableRemote("inout", false);
		app.waitForShutdown();
	}
	catch (std::exception& e)
	{
		cerr << "std::exception:" << e.what() << std::endl;
	}
	catch (...)
	{
		cerr << "unknown exception." << std::endl;
	}
	return -1;
}