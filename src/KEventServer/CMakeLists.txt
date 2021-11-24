
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)

gen_server(tars tarskevent)

target_link_libraries(tarskevent esclient k8sclient ssl)