//
// Created by jarod on 2022/5/5.
//

#include "NodeManager.h"
#include "NodePush.h"
#include "servant/Application.h"

// class AsyncNodePrxCallback : public NodePrxCallback
// {
// public:
// 	AsyncNodePrxCallback(const string &nodeName, CurrentPtr &current) : _nodeName(nodeName), _current(current)
// 	{

// 	}

// 	AsyncNodePrxCallback(const string &nodeName, const ServerStateDesc &desc, CurrentPtr &current) : _nodeName(nodeName), _desc(desc), _current(current)
// 	{

// 	}

// 	virtual void callback_notifyServer(tars::Int32 ret,  const std::string& result)
// 	{
// 		AdminReg::async_response_notifyServer(_current, ret, result);
// 	}
// 	virtual void callback_notifyServer_exception(tars::Int32 ret)
// 	{
// 		AdminReg::async_response_notifyServer(_current, ret, "");
// 	}

// 	virtual void callback_shutdown(tars::Int32 ret,  const std::string& result)
// 	{
// 		AdminReg::async_response_shutdownNode(_current, ret, result);
// 	}

// 	virtual void callback_shutdown_exception(tars::Int32 ret)
// 	{
// 		AdminReg::async_response_shutdownNode(_current, ret, "");
// 	}

// 	virtual void callback_startServer(tars::Int32 ret,  const std::string& result)
// 	{
// 		AdminReg::async_response_startServer(_current, ret, result);
// 	}
// 	virtual void callback_startServer_exception(tars::Int32 ret)
// 	{
// 		AdminReg::async_response_startServer(_current, ret, "");
// 	}

// 	virtual void callback_stopServer(tars::Int32 ret,  const std::string& result)
// 	{
// 		AdminReg::async_response_stopServer(_current, ret, result);
// 	}

// 	virtual void callback_stopServer_exception(tars::Int32 ret)
// 	{
// 		AdminReg::async_response_stopServer(_current, ret, "");
// 	}

// protected:
// 	string 	   _nodeName;
// 	ServerStateDesc	_desc;
// 	CurrentPtr _current;
// };

NodeManager::NodeManager()
{
	_timeoutQueue.setTimeout(_timeout);
}

// NodePrx NodeManager::getNodePrx(const string& nodeName)
// {
// 	{
// 		TC_ThreadLock::Lock lock(_NodePrxLock);

// 		if (_mapNodePrxCache.find(nodeName) != _mapNodePrxCache.end())
// 		{
// 			return _mapNodePrxCache[nodeName];
// 		}
// 	}

// 	NodePrx nodePrx = DbProxy::getInstance()->getNodePrx(nodeName);

// 	TC_ThreadLock::Lock lock(_NodePrxLock);

// 	_mapNodePrxCache[nodeName] = nodePrx;

// 	return nodePrx;
// }

// void NodeManager::eraseNodePrx(const string& nodeName)
// {
// 	TC_ThreadLock::Lock lock(_NodePrxLock);

// 	_mapNodePrxCache.erase(nodeName);
// }

unordered_map<string, NodeManager::UidTimeStr> NodeManager::getNodeList()
{
	TC_ThreadLock::Lock lock(_NodePrxLock);
	return _mapNodeId;
}

void NodeManager::createNodeCurrent(const string& nodeName, const string &sid, CurrentPtr &current)
{

	TC_ThreadLock::Lock lock(_NodePrxLock);

	auto it = _mapNodeId.find(nodeName);
	if(it != _mapNodeId.end())
	{
		it->second.timeStr = TC_Common::now2str("%Y-%m-%d %H:%M:%S");

//		TLOG_DEBUG("nodeName:" << nodeName << ", connection uid size:" << it->second.uids.size() << ", uid:" << current->getUId() << endl);

		auto ii = it->second.its.find(current->getUId());
		if(ii != it->second.its.end())
		{
			it->second.uids.erase(ii->second);
		}

		it->second.uids.push_front(current->getUId());
		it->second.its[current->getUId()] = it->second.uids.begin();
	}
	else
	{
		TLOG_DEBUG("nodeName:" << nodeName << ", uid:" << current->getUId() << endl);

		UidTimeStr &str = _mapNodeId[nodeName];
		str.timeStr = TC_Common::now2str("%Y-%m-%d %H:%M:%S");
		str.uids.push_front(current->getUId());
		str.its[current->getUId()] = str.uids.begin();
	}

	_mapIdNode[current->getUId()].insert(nodeName);
	_mapIdCurrent[current->getUId()] = current;
}

void NodeManager::deleteNodeCurrent(const string& nodeName, const string &sid, CurrentPtr &current)
{
	TC_ThreadLock::Lock lock(_NodePrxLock);

	auto it = _mapNodeId.find(nodeName);

	if(it != _mapNodeId.end())
	{
		//所有连接上, 把这个节点下线
		for(auto uid: it->second.uids)
		{
			auto ii = _mapIdNode.find(uid);
			if(ii != _mapIdNode.end())
			{
				TLOG_DEBUG("nodeName:" << nodeName << ", uid:" << current->getUId() << endl);

				ii->second.erase(nodeName);
			}
		}
		_mapNodeId.erase(it);
	}
}

CurrentPtr NodeManager::getNodeCurrent(const string& nodeName)
{
	TC_ThreadLock::Lock lock(_NodePrxLock);

	auto it = _mapNodeId.find(nodeName);
	if(it != _mapNodeId.end())
	{
		auto uid = it->second.uids.begin();
		if(uid == it->second.uids.end())
		{
			TLOG_ERROR("nodeName:" << nodeName << ", connection uid size:" << it->second.uids.size() << " no alive connection." << endl);
			return  NULL;
		}
//		TLOG_DEBUG("nodeName:" << nodeName << ", connection uid size:" << it->second.uids.size() << ", uid:" << *uid << endl);

		return _mapIdCurrent.at(*uid);
	}

	TLOG_ERROR("nodeName:" << nodeName << " no alive connection." << endl);

	return NULL;
}

void NodeManager::eraseNodeCurrent(CurrentPtr &current)
{
	TC_ThreadLock::Lock lock(_NodePrxLock);

	auto it = _mapIdNode.find(current->getUId());
	if(it != _mapIdNode.end())
	{
		for(auto nodeName : it->second)
		{
			auto ii = _mapNodeId.find(nodeName);
			if(ii != _mapNodeId.end())
			{
				auto iu = ii->second.its.find(current->getUId());
				if(iu != ii->second.its.end())
				{
					ii->second.uids.erase(iu->second);
					ii->second.its.erase(iu);
				}
			}
		}
		_mapIdCurrent.erase(it->first);
		_mapIdNode.erase(it);
	}
}

void NodeManager::terminate()
{
	std::unique_lock<std::mutex> lock(_mutex);

	_terminate = true;

	_cond.notify_one();
}

void NodeManager::run()
{
	std::function<void(NodeResultInfoPtr&)> df = [](NodeResultInfoPtr &ptr){
		ptr->_callback(ptr->_current, true, 0, "");
	};

	while(!_terminate)
	{
		{
			std::unique_lock<std::mutex> lock(_mutex);

			_cond.wait_for(lock, std::chrono::milliseconds(100));
		}

		_timeoutQueue.timeout(df);
	}
}

int NodeManager::reportResult(int requestId, const string &funcName, int ret,  const string &result, CurrentPtr current)
{
	auto ptr = _timeoutQueue.get(requestId, true);

	if(ptr)
	{
		ptr->_ret = ret;
		ptr->_result = result;

		if(!ptr->_current)
		{
			//同步调用, 唤醒
			std::unique_lock<std::mutex> lock(ptr->_m);
			ptr->_c.notify_one();
		}
		else
		{
			//异步调用, 回包
			ptr->_callback(ptr->_current, false, ptr->_ret, ptr->_result);
		}
	}
	else
	{
		TLOG_DEBUG("requestId:" << requestId << ", " << funcName << ", timeout" <<endl);

	}
	return 0;
}

int NodeManager::requestNode(const string & nodeName, string &out, CurrentPtr current, NodeManager::push_type push, NodeManager::callback_type callback)
{
	CurrentPtr nodeCurrent = getNodeCurrent(nodeName);
	if(nodeCurrent)
	{
		NodeResultInfoPtr ptr = new NodeResultInfo(_timeoutQueue.generateId(), current, callback);

		_timeoutQueue.push(ptr, ptr->_requestId);

		if(current)
		{
			//异步
			push(nodeCurrent, ptr->_requestId);
		}
		else
		{
			//同步
			std::unique_lock<std::mutex> lock(ptr->_m);

			push(nodeCurrent, ptr->_requestId);

			if(cv_status::no_timeout == ptr->_c.wait_for(lock, std::chrono::milliseconds(_timeout)))
			{
				out = ptr->_result;
				return ptr->_callback(ptr->_current, false, ptr->_ret, ptr->_result);
			}
			else
			{
				return EM_TARS_CALL_NODE_TIMEOUT_ERR;
			}
		}

		return EM_TARS_SUCCESS;

	}
	else
	{
		
		TLOG_ERROR("nodeName:" << nodeName << ", no long connection" <<endl);
		return EM_TARS_NODE_NO_CONNECTION;
	}
}

int NodeManager::pingNode(const string & nodeName, string &out, tars::CurrentPtr current)
{
	NodeManager::push_type push = [nodeName](CurrentPtr &nodeCurrent, int requestId){
		TLOG_DEBUG("NodeManager::pingNode push name:" << nodeName <<endl);

		NodePush::async_response_push_ping(nodeCurrent, requestId, nodeName);
	};

	NodeManager::callback_type callback = [nodeName](CurrentPtr &current, bool timeout, int ret, const string &buff)
	{
		TLOG_DEBUG("NodeManager::pingNode callback:" << nodeName << ", timeout:" << timeout << ", ret:" << ret << ", result:" << buff <<endl);

		if(current)
		{
			AdminReg::async_response_pingNode(current, timeout?false:true, buff);
		}

		return ret;
	};

	return NodeManager::getInstance()->requestNode(nodeName, out, current, push, callback);
}

int NodeManager::startServer(const string & application, const string & serverName, const string & nodeName, string &out, tars::CurrentPtr current)
{
	NodeManager::push_type push = [application, serverName, nodeName](CurrentPtr &nodeCurrent, int requestId){
		TLOG_DEBUG("NodeManager::startServer push :" << application << "." << serverName << "_" << nodeName <<endl);
		NodePush::async_response_push_startServer(nodeCurrent, requestId, nodeName, application, serverName);
	};

	NodeManager::callback_type callback = [application, serverName, nodeName](CurrentPtr &current, bool timeout, int ret, const string &buff)
	{
		TLOG_DEBUG("NodeManager::startServer: " << application << "." << serverName << "_" << nodeName << ", timeout:" << timeout << ", ret:" << ret << ", result:" << buff <<endl);

		if(current)
		{
			AdminReg::async_response_startServer(current, timeout?EM_TARS_CALL_NODE_TIMEOUT_ERR:ret, buff);
		}

		return ret;
	};

	return requestNode(nodeName, out, current, push, callback);
}

int NodeManager::stopServer(const string & application, const string & serverName, const string & nodeName, string &out, tars::CurrentPtr current)
{
	NodeManager::push_type push = [application, serverName, nodeName](CurrentPtr &nodeCurrent, int requestId){
		TLOG_DEBUG("NodeManager::stopServer push :" << application << "." << serverName << "_" << nodeName <<endl);
		NodePush::async_response_push_stopServer(nodeCurrent, requestId, nodeName, application, serverName);
	};

	NodeManager::callback_type callback = [application, serverName, nodeName](CurrentPtr &current, bool timeout, int ret, const string &buff)
	{
		TLOG_DEBUG("NodeManager::stopServer: " << application << "." << serverName << "_" << nodeName << ", timeout:" << timeout << ", ret:" << ret << ", result:" << buff <<endl);

		if(current)
		{
			AdminReg::async_response_stopServer(current, timeout?EM_TARS_CALL_NODE_TIMEOUT_ERR:ret, buff);
		}

		return ret;
	};

	return requestNode(nodeName, out, current, push, callback);
}

int NodeManager::notifyServer(const string & application, const string & serverName, const string & nodeName, const string &command, string & out, tars::CurrentPtr current)
{
	NodeManager::push_type push = [application, serverName, nodeName, command](CurrentPtr &nodeCurrent, int requestId){
		TLOG_DEBUG("NodeManager::notifyServer push :" << application << "." << serverName << "_" << nodeName <<endl);
		NodePush::async_response_push_notifyServer(nodeCurrent, requestId, nodeName, application, serverName, command);
	};

	NodeManager::callback_type callback = [application, serverName, nodeName](CurrentPtr &current, bool timeout, int ret, const string &buff)
	{
		TLOG_DEBUG("NodeManager::notifyServer: " << application << "." << serverName << "_" << nodeName << ", timeout:" << timeout << ", ret:" << ret << ", result:" << buff <<endl);

		if(current)
		{
			AdminReg::async_response_notifyServer(current, timeout?EM_TARS_CALL_NODE_TIMEOUT_ERR:ret, buff);
		}

		return ret;
	};

	return requestNode(nodeName, out, current, push, callback);
}
