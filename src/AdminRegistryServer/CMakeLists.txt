
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)

gen_server(tars tarsadminregistry)

target_link_libraries(tarsadminregistry  k8sclient ssl)