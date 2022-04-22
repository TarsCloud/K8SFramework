
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)

gen_server(tars tarsnotify)

target_link_libraries(tarsnotify esclient k8sclient ssl)
