
include_directories(../ThirdParty/asio/include)
include_directories(../ThirdParty/rapidjson/include)


aux_source_directory(. DIR_SRC)

add_library(k8sclient ${DIR_SRC})
