#pragma once

#include "TCodec.h"
#include "TServant.h"

struct TEndpoint
{
    int asyncThread;;
    std::string resourceName;
    std::string templateName;
    std::string profileContent;
    std::vector <TServant> servants;
    std::set <std::string> activatedPods;
    std::set <std::string> inActivatedPods;
};

DECODE_JSON_TO_STRUCT(TEndpoint, document)
{
    TEndpoint te{};
    auto&& tars = document.at_pointer("/spec/tars");
    READ_FROM_JSON(te.asyncThread, tars.at("asyncThread"));
    READ_FROM_JSON(te.templateName, tars.at("template"));
    READ_FROM_JSON(te.profileContent, tars.at("profile"));
    READ_FROM_JSON(te.servants, tars.at("servants"));

    READ_FROM_JSON(te.resourceName, document.at_pointer("/metadata/name"));

    auto&& pods = document.at_pointer("/status/pods");
    for (auto&& pod: pods.get_array())
    {
        VAR_FROM_JSON(std::string, settingState, pod.at("settingState"));
        VAR_FROM_JSON(std::string, presentState, pod.at("presentState"));
        VAR_FROM_JSON(std::string, podName, pod.at("name"));
        if (presentState == "Active" && settingState == "Active")
        {
            te.activatedPods.insert(podName);
        }
        else
        {
            te.inActivatedPods.insert(podName);
        }
    }
    return te;
}
