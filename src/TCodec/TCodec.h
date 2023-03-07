#pragma once

#include <boost/json.hpp>

constexpr char API_GROUP_VERSION[] = "k8s.tars.io/v1beta3";

#define READ_FROM_JSON(v, p) (v)=boost::json::value_to<decltype(v)>(p)
#define VAR_FROM_JSON(t, v, p) auto v = boost::json::value_to<t>(p)

#define DECODE_JSON_TO_STRUCT(T, j) inline T tag_invoke(boost::json::value_to_tag<T>, boost::json::value const& j)
#define ENCODE_STRUCT_TO_JSON(T, s, j) inline void tag_invoke(boost::json::value_from_tag, boost::json::value& j, T const& s)


