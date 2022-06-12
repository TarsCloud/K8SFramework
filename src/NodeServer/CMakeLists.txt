include_directories(../RegistryServer)
include_directories(../AdminRegistryServer)

gen_server(tars tarsnode)

add_dependencies(tarsnode tarsregistry tarsadminregistry)

target_link_libraries(tarsnode )
