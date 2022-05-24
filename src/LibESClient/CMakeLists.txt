
include_directories(../ThirdParty/asio/include)

aux_source_directory(. DIR_SRC)

add_library(esclient ${DIR_SRC})
