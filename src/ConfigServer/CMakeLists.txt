
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)

gen_server(tars tarsconfig)

target_link_libraries(tarsconfig  k8sclient ssl)
