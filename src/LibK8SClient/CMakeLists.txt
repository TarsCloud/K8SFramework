
aux_source_directory(. DIR_SRC)

add_library(k8sclient ${DIR_SRC})

target_link_libraries(k8sclient crypto ssl)
