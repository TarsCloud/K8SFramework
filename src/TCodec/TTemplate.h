#pragma once

#include "TCodec.h"

struct TTemplate
{
    std::string name;
    std::string parent;
    std::string content;
};

DECODE_JSON_TO_STRUCT(TTemplate, document)
{
    TTemplate tt{};
    READ_FROM_JSON(tt.name, document.at_pointer("/metadata/name"));
    READ_FROM_JSON(tt.parent, document.at_pointer("/spec/parent"));
    READ_FROM_JSON(tt.content, document.at_pointer("/spec/content"));
    return tt;
}
