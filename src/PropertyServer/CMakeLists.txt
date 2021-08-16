
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)

gen_server(tars tarsproperty)
target_link_libraries(tarsproperty  elsclient)
