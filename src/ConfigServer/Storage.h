#include <K8SWatcher.h>
#include <unordered_map>

class Storage
{
public:

    static void getSeqMap(const std::function<void(const std::unordered_map<std::string, int>& seqMap)>&);

    static void prePodList();

    static void postPodList();

    static void onPodAdded(const boost::json::value& v, K8SWatchEventDrive drive);

    static void onPodDelete(const boost::json::value& v);

    static void onPodModified(const boost::json::value& v);
};
