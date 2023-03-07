#pragma once

#include "TCodec.h"
#include "TServant.h"

struct UPChain
{
    std::string resourceName{};
    std::vector<tars::EndpointF> defaults{};
    std::map<std::string, std::vector<tars::EndpointF>> customs{};
};

namespace tars
{
    DECODE_JSON_TO_STRUCT(tars::EndpointF, j)
    {
        auto tf = tars::EndpointF{};
        READ_FROM_JSON(tf.host, j.at("host"));
        READ_FROM_JSON(tf.port, j.at("port"));
        bool isTcp{ true};
        READ_FROM_JSON(isTcp, j.at("isTcp"));
        tf.istcp=isTcp;
        READ_FROM_JSON(tf.timeout, j.at("timeout"));
        return tf;
    }
}


DECODE_JSON_TO_STRUCT(UPChain, j)
{
    UPChain upChain{};
    READ_FROM_JSON(upChain.resourceName, j.at_pointer("/metadata/name"));
    auto&& pUpChain = j.at_pointer("/upChain");
    assert(pUpChain.is_object());
    for (auto&& item: pUpChain.get_object())
    {
        auto key = item.key();
        if (key == "default")
        {
            READ_FROM_JSON(upChain.defaults, item.value());
        }
        else
        {
            std::vector<tars::EndpointF> v{};
            READ_FROM_JSON(v, item.value());
            upChain.customs[key].swap(v);
        }
    }
    return upChain;
}
