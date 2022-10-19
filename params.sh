#!/usr/bin/env sh

# shellcheck disable=SC2034

_REGISTRY_URL_=""
_REGISTRY_USER_=""
_REGISTRY_PASSWORD_=""

_CHART_VERSION_=1.3.0-master
_CHART_APPVERSION_=v1beta3
_CHART_DST_=charts

_BUILD_VERSION_=v1.3.0-master

#########################################################################################################################
### Please Do Not Edit This Block Unless You Know What You Are Doing ###
_TARS_CPP_DIR_=submodule/TarsCpp
_TARS_WEB_DIR_=submodule/TarsWeb

_BASES_="cppbase javabase nodejsbase php74base"

_CONTROLLER_SERVERS_="tarscontroller tarsagent"

_FRAMEWORK_SERVERS_="tarskaniko tarsimage  tarsnode tarsAdminRegistry tarsregistry tarsconfig tarslog tarsnotify tarsstat tarsproperty \
                     tarsquerystat tarsqueryproperty tarskevent tarsweb elasticsearch"

_CRD_SERVED_VERSIONS_="v1beta2 v1beta3"
_CRD_STORAGE_VERSION_="v1beta3"

_PARAMS_="TARS_CPP_DIR TARS_WEB_DIR                                                                                  \
          REGISTRY_URL REGISTRY_USER REGISTRY_PASSWORD                                                               \
          BASES CONTROLLER_SERVERS FRAMEWORK_SERVERS                                                                 \
          CRD_SERVED_VERSIONS CRD_STORAGE_VERSION                                                                    \
          BUILD_VERSION PLATFORMS                                                                                    \
          CHART_VERSION CHART_APPVERSION CHART_DST                                                                    
          "

_PLATFORMS_="linux/amd64"
_SUPPORT_PLATFORMS_="linux/amd64 linux/386 linux/arm64 linux/riscv64 linux/ppc64le linux/s390x"

PARAM=$(echo "$1" | tr "[:lower:]" "[:upper:]")
eval "VALUE=\$_${PARAM}_"
echo "${VALUE}"

### Please Do Not Edit This Block Unless You Know What You Are Doing ###
#########################################################################################################################
