#include <K8SWatcher.h>
#include <unordered_map>

class Storage
{
public:

	static void getPodIPMap(const std::function<void(const std::unordered_map <std::string, std::string>& seqMap)>&);

	static void prePodList();

	static void postPodList();

	static void onPodAdded(const rapidjson::Value& v, K8SWatchEventDrive drive);

	static void onPodDelete(const rapidjson::Value& v);

	static void onPodModified(const rapidjson::Value& v);
};