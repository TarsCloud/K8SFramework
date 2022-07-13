
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)
include_directories(../NodeServer)
include_directories(../RegistryServer)

gen_server(tars tarsadminregistry)

add_dependencies(tarsadminregistry tarsnode)
