#include "QueryImp.h"
#include "ServerInfoInterface.h"
#include "servant/RemoteLogger.h"

#include <string>

string eFunToStr(const FUNID eFnId) {
    string sFun{};
    switch (eFnId) {
        case FUNID_findObjectByIdInSameGroup: {
            sFun = "findObjectByIdInSameGroup";
        }
            break;
        case FUNID_findObjectByIdInSameSet: {
            sFun = "findObjectByIdInSameSet";
        }
            break;
        case FUNID_findObjectById4Any: {
            sFun = "findObjectById4All";
        }
            break;
        case FUNID_findObjectById: {
            sFun = "findObjectById";
        }
            break;
        case FUNID_findObjectById4All: {
            sFun = "findObjectById4All";
        }
            break;
        case FUNID_findObjectByIdInSameStation: {
            sFun = "findObjectByIdInSameStation";
        }
            break;
        default:
            sFun = "UNKNOWN";
            break;
    }
    return sFun;
}

static void findObjectById_(const std::string &id, vector <EndpointF> *activeEp, vector <EndpointF> *inactiveEp) {
    ServerInfoInterface::instance().findEndpoint(id, activeEp, inactiveEp);
}

void QueryImp::initialize() {
}

vector <EndpointF> QueryImp::findObjectById(const std::string &id, CurrentPtr current) {
    vector <EndpointF> activeEp;
    findObjectById_(id, &activeEp, nullptr);

    std::ostringstream os;
    doDaylog(FUNID_findObjectById, id, activeEp, vector<EndpointF>(), current, os);

    return activeEp;
}

Int32 QueryImp::findObjectById4Any(const std::string &id, vector <EndpointF> &activeEp, vector <EndpointF> &inactiveEp, CurrentPtr current) {
    findObjectById_(id, &activeEp, &inactiveEp);

    std::ostringstream os;
    doDaylog(FUNID_findObjectById4Any, id, activeEp, inactiveEp, current, os);

    return 0;
}

int QueryImp::findObjectById4All(const std::string &id, vector <EndpointF> &activeEp, vector <EndpointF> &inactiveEp, CurrentPtr current) {
    findObjectById_(id, &activeEp, &inactiveEp);

    std::ostringstream os;
    doDaylog(FUNID_findObjectById4All, id, activeEp, inactiveEp, current, os);

    return 0;
}

int QueryImp::findObjectByIdInSameGroup(const std::string &id, vector <EndpointF> &activeEp, vector <EndpointF> &inactiveEp, CurrentPtr current) {
    findObjectById_(id, &activeEp, &inactiveEp);

    std::ostringstream os;
    doDaylog(FUNID_findObjectByIdInSameGroup, id, activeEp, inactiveEp, current, os);

    return 0;
}

Int32 QueryImp::findObjectByIdInSameStation(const std::string &id, const std::string &sStation, vector <EndpointF> &activeEp, vector <EndpointF> &inactiveEp,
                                            CurrentPtr current) {
    findObjectById_(id, &activeEp, &inactiveEp);

    std::ostringstream os;
    doDaylog(FUNID_findObjectByIdInSameStation, id, activeEp, inactiveEp, current, os);

    return 0;
}

Int32
QueryImp::findObjectByIdInSameSet(const std::string &id, const std::string &setId, vector <EndpointF> &activeEp,
                                  vector <EndpointF> &inactiveEp,
                                  CurrentPtr current) {
    findObjectById_(id, &activeEp, &inactiveEp);

    std::ostringstream os;
    doDaylog(FUNID_findObjectByIdInSameSet, id, activeEp, inactiveEp, current, os);

    return 0;
}

void
QueryImp::doDaylog(FUNID eFnId, const string &id, const vector <EndpointF> &activeEp,
                   const vector <EndpointF> &inactiveEp, const CurrentPtr &current, const ostringstream &os,
                   const string &sSetid) {
    string sEpList;

    for (size_t i = 0; i < activeEp.size(); i++) {
        if (0 != i) {
            sEpList += ";";
        }
        sEpList += activeEp[i].host + ":" + TC_Common::tostr(activeEp[i].port);
    }

    sEpList += "|";

    for (size_t i = 0; i < inactiveEp.size(); i++) {
        if (0 != i) {
            sEpList += ";";
        }
        sEpList += inactiveEp[i].host + ":" + TC_Common::tostr(inactiveEp[i].port);
    }

    switch (eFnId) {
        case FUNID_findObjectById4All:
        case FUNID_findObjectByIdInSameGroup: {
            FDLOG("query_idc") << eFunToStr(eFnId) << "|" << current->getIp() << "|" << current->getPort() << "|" << id << "|" << sSetid << "|" << sEpList << os.str() << endl;
        }
            break;
        case FUNID_findObjectByIdInSameSet: {
            FDLOG("query_set") << eFunToStr(eFnId) << "|" << current->getIp() << "|" << current->getPort() << "|" << id << "|" << sSetid << "|" << sEpList << os.str() << endl;
        }
            break;
        case FUNID_findObjectById4Any:
        case FUNID_findObjectById:
        case FUNID_findObjectByIdInSameStation:
        default: {
            FDLOG("query") << eFunToStr(eFnId) << "|" << current->getIp() << "|" << current->getPort() << "|" << id << "|" << sSetid << "|" << sEpList << os.str() << endl;
        }
            break;
    }
}
