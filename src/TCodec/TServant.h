#pragma once

#include "TCodec.h"

struct TServant
{
    bool isTcp{ true };
    bool isTars{ true };
    int port{};
    int32_t thread{ 3 };
    int32_t connection{ 10000 };
    int32_t timeout{ 60000 };
    int32_t capacity{ 10000 };
    std::string name;
};

DECODE_JSON_TO_STRUCT(TServant, j)
{
    TServant ts{};
    READ_FROM_JSON(ts.isTcp, j.at("isTcp"));
    READ_FROM_JSON(ts.isTars, j.at("isTars"));
    READ_FROM_JSON(ts.port, j.at("port"));
    READ_FROM_JSON(ts.thread, j.at("thread"));
    READ_FROM_JSON(ts.connection, j.at("connection"));
    READ_FROM_JSON(ts.timeout, j.at("timeout"));
    READ_FROM_JSON(ts.capacity, j.at("capacity"));
    READ_FROM_JSON(ts.name, j.at("name"));
    return ts;
}

ENCODE_STRUCT_TO_JSON(TServant, s, j)
{
    j = boost::json::object{
            { "name",       s.name },
            { "port",       s.port },
            { "isTcp",      s.isTcp },
            { "isTars",     s.isTars },
            { "thread",     s.thread },
            { "capacity",   s.capacity },
            { "connection", s.connection },
    };
}
