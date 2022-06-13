
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)

gen_server(tars tarsAdminRegistry)

target_link_libraries(tarsAdminRegistry  k8sclient ssl)
