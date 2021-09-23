#include "NotifyMsgQueue.h"
#include "servant/Application.h"

void NotifyMsgQueue::init() {
    extern TC_Config *g_pconf;

    _protocol = g_pconf->get("/tars/elk<protocol>", "http");
    initElkTupleNodes(*g_pconf);

    _indexPre = g_pconf->get("/tars/elk<indexPre>", "notify_");

    _ast.setTimeout(10000);
    _ast.start();

    TLOG_DEBUG("protocol:" << _protocol << ", indexPre:" << _indexPre << endl);

    initLimit(g_pconf);

    start();
}

void NotifyMsgQueue::terminate() {
    _terminate = true;

    TC_ThreadLock::Lock lock(*this);
    notifyAll();
}

void NotifyMsgQueue::add(const NotifyRecord &notifyRecord) {
    _qMsg.push_back(notifyRecord);
}

void NotifyMsgQueue::run() {
    while (!_terminate) {
        try {
            string writeToESDate;
            vector<NotifyRecord> vQData;

            do {
                NotifyRecord data;
                _qMsg.pop_front(data, -1);

                if (!checkLimit(data.app + "." + data.server)) {
                    TLOG_ERROR("limit fail|" << data.app << "." << data.server << "|" << data.podName << "|" << data.level << "|" << data.message << endl);
                    continue;
                }
                if (writeToESDate.empty()) {
                    writeToESDate = TC_Common::tm2str(data.notifyTime, "%Y%m%d");
                }

                string notifyDate = TC_Common::tm2str(data.notifyTime, "%Y%m%d");
                if (writeToESDate != notifyDate) {
                    _qMsg.push_front(data, false);
                    break;
                }

                vQData.push_back(data);
            } while ((!_qMsg.empty()) && (vQData.size() < 500));

            writeToES(vQData, writeToESDate);
        }
        catch (exception &ex) {
            TLOG_ERROR("exception:" << ex.what() << endl);
        }
        catch (...) {
            TLOG_ERROR("exception:e unknown error." << endl);
        }
    }
}

void NotifyMsgQueue::initElkTupleNodes(const tars::TC_Config &config) {
    _elkTupleNodes.clear();

    vector<string> elkNodes = config.getDomainKey("/tars/elk/nodes");
    if (elkNodes.empty()) {
        TLOGERROR("NotifyMsgQueue::initialize empty elk nodes " << endl);
        exit(0);
    }

    for (auto &item : elkNodes) {
        vector<string> vOneNode = TC_Common::sepstr<string>(item, ":", true);
        if (vOneNode.size() < 2) {
            TLOGERROR("NotifyMsgQueue::initialize wrong elk nodes:" << item << endl);
            continue;
        }
        _elkTupleNodes.emplace_back(vOneNode[0], std::stoi(vOneNode[1]));
    }

    std::srand(std::time(nullptr));

    TLOG_DEBUG("NotifyMsgQueue::initialize get elk nodes size:" << _elkTupleNodes.size() << endl);
}

string NotifyMsgQueue::getELKNodeAddress() {
    if (_elkTupleNodes.empty()) {
        throw std::runtime_error(
                std::string("fatal error: empty elk node addresses"));
    }

    auto tuple = _elkTupleNodes[std::rand() % _elkTupleNodes.size()];
    return _protocol + "://" + std::get<0>(tuple) + ":" + std::to_string(std::get<1>(tuple));
}

string NotifyMsgQueue::writeToJson(const NotifyRecord &record) {
    tars::JsonValueObjPtr p = new tars::JsonValueObj();
    p->value.insert(make_pair("f_timestamp", tars::JsonOutput::writeJson(TNOW)));
    p->value.insert(make_pair("notifyTime", tars::JsonOutput::writeJson(TC_Common::tm2str(record.notifyTime, "%FT%T%z"))));
    p->value.insert(make_pair("app", tars::JsonOutput::writeJson(record.app)));
    p->value.insert(make_pair("server", tars::JsonOutput::writeJson(record.server)));
    p->value.insert(make_pair("podName", tars::JsonOutput::writeJson(record.podName)));
    p->value.insert(make_pair("impThread", tars::JsonOutput::writeJson(record.impThread)));
    p->value.insert(make_pair("level", tars::JsonOutput::writeJson(record.level)));
    p->value.insert(make_pair("message", tars::JsonOutput::writeJson(record.message)));
    p->value.insert(make_pair("source", tars::JsonOutput::writeJson(record.source)));
    return tars::TC_Json::writeValue(p);
}

class AsyncHttpCallback: public TC_HttpAsync::RequestCallback {
public:
    AsyncHttpCallback() {
    }
    virtual bool onContinue(TC_HttpResponse &stHttpResponse) { return true; }
    virtual void onSucc(TC_HttpResponse &stHttpResponse) {
        TLOG_DEBUG("response succ" << endl);
    }
    virtual void onFailed(FAILED_CODE ret, const string &info) {
        TLOG_ERROR("ret: " << ret << ", info:" << info << endl);
    }
    virtual void onClose() {
    }
protected:
};

void NotifyMsgQueue::writeToES(const vector<NotifyRecord> &data, const string &date) {
    if (data.empty()) {
        return;
    }

    string esUrl = getELKNodeAddress();
    if (esUrl.empty()) {
        return;
    }

    string index = _indexPre + "_" + date;
    string url = esUrl + "/" + index + "/_bulk?pretty&refresh";

    TLOG_DEBUG("url:" << url << ", size:" << data.size() << endl);

    string buff;
    for (auto record : data) {
        buff += "{\"index\":{}}\r\n";
        buff += writeToJson(record) + "\r\n";
    }

    TC_HttpRequest stHttpReq;
    stHttpReq.setPostRequest(url, buff);
    stHttpReq.setContentType("application/json");

    TC_HttpAsync::RequestCallbackPtr p = new AsyncHttpCallback();

    _ast.doAsyncRequest(stHttpReq, p);
}

void FreqLimit::initLimit(TC_Config *conf) {
    string limitConf = conf->get("/tars/server<notify_limit>", "300:5");
    vector<int> vi = TC_Common::sepstr<int>(limitConf, ":,|");
    if (vi.size() != 2) {
        _interval = 300;
        _count = 5;
    } else {
        _interval = (unsigned int) vi[0];
        _count = vi[1];
        if (_count <= 1) {
            _count = 1;
        }
    }
}

bool FreqLimit::checkLimit(const string &sServer) {
    auto it = _limit.find(sServer);
    time_t t = TNOW;
    if (it != _limit.end()) {
        if (t > _limit[sServer].t + _interval) {
            _limit[sServer].t = t;
            _limit[sServer].count = 1;
            return true;
        } else if (_limit[sServer].count >= _count) {
            return false;
        } else {
            _limit[sServer].count++;
            return true;
        }
    } else {
        LimitData ld;
        ld.t = t;
        ld.count = 1;
        _limit[sServer] = ld;
        return true;
    }
}
