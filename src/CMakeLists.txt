cmake_minimum_required(VERSION 2.8)

project(TarsK8S)

option(TARS_MYSQL "option for mysql" ON)
option(TARS_SSL "option for ssl" ON)
option(TARS_GMSSL "option for ssl" OFF)
option(TARS_HTTP2 "option for http2" OFF)

set(TARS_TOOL_FLAG "--with-tars")

#set(CMAKE_BUILD_TYPE "Debug")
set(CMAKE_BUILD_TYPE "Release")

set(TARS_WEB_HOST "http://tars.test.com")

if (WIN32)
    include(c:\\tars\\cpp\\makefile\\tars-tools.cmake)
else ()
    include(/usr/local/tars/cpp/makefile/tars-tools.cmake)
endif ()

include_directories(/usr/local/tars/cpp/thirdparty/include)
link_directories(/usr/local/tars/cpp/thirdparty/lib)

include_directories(/usr/local/tars/cpp/include/servant/)

add_subdirectory(LibK8SClient)
add_subdirectory(LibESClient)

include_directories(./LibK8SClient/)
include_directories(./LibESClient/)
include_directories(./TCodec/)
add_subdirectory(ConfigServer)
add_subdirectory(NodeServer)
add_subdirectory(NotifyServer)
add_subdirectory(AdminRegistryServer)
add_subdirectory(RegistryServer)
add_subdirectory(PropertyServer)
add_subdirectory(StatServer)
add_subdirectory(QueryServer)
add_subdirectory(LogServer)
add_subdirectory(KEventServer)
#add_subdirectory(test)

############################################################################

set(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "${CMAKE_SOURCE_DIR}/cmake")

include(cmake/golang.cmake)

add_subdirectory(ImageServer)
add_subdirectory(AgentServer)
add_subdirectory(ControllerServer)
add_subdirectory(KanikoServer)
add_subdirectory(WebhookServer)

install(DIRECTORY ../builder/deploy DESTINATION ${TARS_PATH}
        PATTERN "*.sh"
        PERMISSIONS OWNER_EXECUTE OWNER_WRITE OWNER_READ GROUP_EXECUTE GROUP_READ)
