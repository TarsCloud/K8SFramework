include_directories(../RegistryServer)

gen_server(tars tarsnode)

add_dependencies(tarsnode tarsregistry)

target_link_libraries(tarsnode -static-libgcc -static-libstdc++)
