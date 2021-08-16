
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)

gen_server(tars tarsregistry)

target_link_libraries(tarsregistry  k8sclient ssl)
