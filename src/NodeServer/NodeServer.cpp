#include "NodeServer.h"
#include "NodeImp.h"
#include "ServerImp.h"
#include "ServerObject.h"
#include "ProxyManger.h"
#include "Container.h"

void NodeServer::initialize()
{
	if (!container::loadValues())
	{
		TLOGERROR("loadValues error" << std::endl;);
		std::cout << "loadValues error";
		exit(-1);
	}

	try
	{
		addServant<NodeImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".NodeObj");
		addServant<ServerImp>(ServerConfig::Application + "." + ServerConfig::ServerName + ".ServerObj");

		auto registryPrx = ProxyManger::instance().getRegistryProxy();
		registryPrx->updateServerState(container::podName, etos(Active), etos(Activating));

		ServerObject::startPatrol();
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
