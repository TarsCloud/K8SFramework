#!/usr/bin/env bash

_HOST_NAME_=${HOSTNAME}
if [ -z "$_HOST_NAME_" ]; then
  echo "got empty [HOSTNAME] env value"
  exit 255
fi

_K8S_POD_NAME_=${PodName}
if [ -z "$_K8S_POD_NAME_" ]; then
  echo "got empty [PodName] env value"
  exit 255
fi

_K8S_POD_IP_=${PodIP}
if [ -z "$_K8S_POD_IP_" ]; then
  echo "got empty [PodIP] env value"
  exit 255
fi

_TARS_SERVER_APP_=${ServerApp}
if [ -z "$_TARS_SERVER_APP_" ]; then
  echo "got empty [ServerApp] env value"
  exit 255
fi

_TARS_SERVER_NAME_=${ServerName}
if [ -z "$_TARS_SERVER_NAME_" ]; then
  echo "got empty [ServerName] env value"
  exit 255
fi

declare -l _LISTEN_ADDRESS_=${_K8S_POD_NAME_}.${_TARS_SERVER_APP_}-${_TARS_SERVER_NAME_}

_IMAGE_BIND_TARSNODE_EXECUTION_FILE_="/tarsnode/bin/tarsnode"
_IMAGE_BIND_TARSNODE_CONF_FILE_="/tarsnode/conf/tarsnode.conf"
_IMAGE_BIND_UTIL_ENTRYPOINT_FILE_="/tarsnode/util/start.sh"
_IMAGE_BIND_UTIL_TIMEZONE_FILE_="/tarsnode/util/timezone.sh"

_TARSNODE_BASE_DIR_="/usr/local/app/tars/tarsnode"

_TARSNODE_BIN_DIR_="${_TARSNODE_BASE_DIR_}/bin"

_TARSNODE_EXECUTION_FILE_="${_TARSNODE_BIN_DIR_}/tarsnode"

_TARSNODE_CONF_DIR_="${_TARSNODE_BASE_DIR_}/conf"

_TARSNODE_CONF_FILE_="${_TARSNODE_CONF_DIR_}/tarsnode.conf"

_TARSNODE_DATA_DIR_="${_TARSNODE_BASE_DIR_}/data"

_TARSNODE_LOG_DIR_="/usr/local/app/tars/app_log"

_TARSNODE_UTIL_DIR_="${_TARSNODE_BASE_DIR_}/util"
_TARSNODE_UTIL_ENTRYPOINT_="${_TARSNODE_UTIL_DIR_}/start.sh"
_TARSNODE_UTIL_TIMEZONE_="${_TARSNODE_UTIL_DIR_}/timezone.sh"

_TARSNODE_ENVIRONMENT_="${_TARSNODE_BASE_DIR_}/util/environment"

if ! mkdir -p ${_TARSNODE_BIN_DIR_}; then
  echo "mkdir -p ${_TARSNODE_BIN_DIR_} error"
  exit 255
fi

if ! mkdir -p ${_TARSNODE_CONF_DIR_}; then
  echo "mkdir -p ${_TARSNODE_CONF_DIR_} error"
  exit 255
fi

if ! mkdir -p ${_TARSNODE_UTIL_DIR_}; then
  echo "mkdir -p ${_TARSNODE_UTIL_DIR_} error"
  exit 255
fi

if ! cp -r ${_IMAGE_BIND_TARSNODE_EXECUTION_FILE_} ${_TARSNODE_EXECUTION_FILE_}; then
  echo "cp -r ${_IMAGE_BIND_TARSNODE_EXECUTION_FILE_} ${_TARSNODE_EXECUTION_FILE_} error"
  exit 255
fi

if ! cp -r ${_IMAGE_BIND_TARSNODE_CONF_FILE_} ${_TARSNODE_CONF_FILE_}; then
  echo "cp -r ${_IMAGE_BIND_TARSNODE_CONF_FILE_} ${_TARSNODE_CONF_FILE_} error"
  exit 255
fi

if ! cp -r ${_IMAGE_BIND_UTIL_ENTRYPOINT_FILE_} ${_TARSNODE_UTIL_ENTRYPOINT_}; then
  echo "cp -r ${_IMAGE_BIND_UTIL_ENTRYPOINT_FILE_} ${_TARSNODE_UTIL_ENTRYPOINT_} error"
  exit 255
fi

if ! cp -r ${_IMAGE_BIND_UTIL_TIMEZONE_FILE_} ${_TARSNODE_UTIL_TIMEZONE_}; then
  echo "cp -r ${_IMAGE_BIND_UTIL_TIMEZONE_FILE_} ${_TARSNODE_UTIL_TIMEZONE_} error"
  exit 255
fi

declare -a ReplaceKeyList=(
  _LISTEN_ADDRESS_
  _TARSNODE_BIN_DIR_
  _TARSNODE_DATA_DIR_
  _TARSNODE_LOG_DIR_
)

declare -a ReplaceFileList=(
  "${_TARSNODE_CONF_FILE_}"
)

for KEY in "${ReplaceKeyList[@]}"; do
  for FILE in "${ReplaceFileList[@]}"; do
    if ! sed -i "s#${KEY}#${!KEY}#g" "${FILE}"; then
      exit 255
    fi
  done
done

export ServerBaseDir=${_TARSNODE_DATA_DIR_}/"${_TARS_SERVER_APP_}"."${_TARS_SERVER_NAME_}"
export ServerBinDir=${ServerBaseDir}/bin
export ServerDataDir=${ServerBaseDir}/data
export ServerConfDir=${ServerBaseDir}/conf
export ServerConfFile=${ServerConfDir}/${_TARS_SERVER_APP_}"."${_TARS_SERVER_NAME_}.config.conf
export ServerLogDir=${_TARSNODE_LOG_DIR_}

ListenAddress=${_LISTEN_ADDRESS_}

declare -a ForwardENVKeyList=(
  PodName
  PodIP
  ListenAddress
  ServerApp
  ServerName
  ServerBaseDir
  ServerBinDir
  ServerConfDir
  ServerDataDir
  ServerLogDir
  ServerConfFile
)

echo -e "#! /usr/bin/env bash" >${_TARSNODE_ENVIRONMENT_}
for KEY in "${ForwardENVKeyList[@]}"; do
  echo -e "export ${KEY}=${!KEY}" >>${_TARSNODE_ENVIRONMENT_}
done
