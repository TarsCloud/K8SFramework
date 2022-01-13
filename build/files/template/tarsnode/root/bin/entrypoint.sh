#!/usr/bin/env bash

declare -a ExpectENVKeyList=(
  HOSTNAME
  PodName
  PodIP
  ServerApp
  ServerName
)

for KEY in "${ExpectENVKeyList[@]}"; do
  if [ -z "${!KEY}" ]; then
    echo "got empty [${KEY}] env value"
    exit 255
  fi
done

declare -l _LISTEN_ADDRESS_=${PodName}.${ServerApp}-${ServerName}

_IMAGE_BIND_TARSNODE_EXECUTION_FILE_="/tarsnode/bin/tarsnode"
_IMAGE_BIND_FOREGROUND_TARSNODE_CONF_FILE_="/tarsnode/conf/foreground.conf"
_IMAGE_BIND_BACKGROUND_TARSNODE_CONF_FILE_="/tarsnode/conf/background.conf"
_IMAGE_BIND_UTIL_START_FILE_="/tarsnode/util/start.sh"
_IMAGE_BIND_UTIL_TIMEZONE_FILE_="/tarsnode/util/timezone.sh"

_TARSNODE_BASE_DIR_="/usr/local/app/tars/tarsnode"

_TARSNODE_BIN_DIR_="${_TARSNODE_BASE_DIR_}/bin"

_TARSNODE_EXECUTION_FILE_="${_TARSNODE_BIN_DIR_}/tarsnode"

_TARSNODE_CONF_DIR_="${_TARSNODE_BASE_DIR_}/conf"

_TARSNODE_CONF_FILE_="${_TARSNODE_CONF_DIR_}/tarsnode.conf"

_TARSNODE_DATA_DIR_="${_TARSNODE_BASE_DIR_}/data"

_TARSNODE_LOG_DIR_="/usr/local/app/tars/app_log"

_TARSNODE_UTIL_DIR_="${_TARSNODE_BASE_DIR_}/util"
_TARSNODE_UTIL_START_="${_TARSNODE_UTIL_DIR_}/start.sh"
_TARSNODE_UTIL_TIMEZONE_="${_TARSNODE_UTIL_DIR_}/timezone.sh"

_TARSNODE_ENVIRONMENT_="${_TARSNODE_UTIL_DIR_}/environment"

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

if ! cp -r ${_IMAGE_BIND_UTIL_START_FILE_} ${_TARSNODE_UTIL_START_}; then
  ec*-+++++\
  o "cp -r ${_IMAGE_BIND_UTIL_START_FILE_} ${_TARSNODE_UTIL_START_} error"
  exit 255
fi

if ! cp -r ${_IMAGE_BIND_UTIL_TIMEZONE_FILE_} ${_TARSNODE_UTIL_TIMEZONE_}; then
  echo "cp -r ${_IMAGE_BIND_UTIL_TIMEZONE_FILE_} ${_TARSNODE_UTIL_TIMEZONE_} error"
  exit 255
fi

case ${LauncherType} in
foreground)
  export ServerLauncherType="foreground"
  if ! cp -r ${_IMAGE_BIND_FOREGROUND_TARSNODE_CONF_FILE_} ${_TARSNODE_CONF_FILE_}; then
    echo "cp -r ${_IMAGE_BIND_FOREGROUND_TARSNODE_CONF_FILE_} ${_TARSNODE_CONF_FILE_} error"
    exit 255
  fi
  ;;
background)
  export ServerLauncherType="background"
  if ! cp -r ${_IMAGE_BIND_BACKGROUND_TARSNODE_CONF_FILE_} ${_TARSNODE_CONF_FILE_}; then
    echo "cp -r ${_IMAGE_BIND_BACKGROUND_TARSNODE_CONF_FILE_} ${_TARSNODE_CONF_FILE_} error"
    exit 255
  fi
  ;;
*)
  export ServerLauncherType="background"
  if ! cp -r ${_IMAGE_BIND_BACKGROUND_TARSNODE_CONF_FILE_} ${_TARSNODE_CONF_FILE_}; then
    echo "cp -r ${_IMAGE_BIND_BACKGROUND_TARSNODE_CONF_FILE_} ${_TARSNODE_CONF_FILE_} error"
    exit 255
  fi
  ;;
esac

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

export ListenAddress=${_LISTEN_ADDRESS_}

export ServerBaseDir=${_TARSNODE_DATA_DIR_}/"${ServerApp}"."${ServerName}"
export ServerBinDir=${ServerBaseDir}/bin
export ServerDataDir=${ServerBaseDir}/data
export ServerConfDir=${ServerBaseDir}/conf
export ServerLogDir=${_TARSNODE_LOG_DIR_}
export ServerConfFile=${ServerConfDir}/${ServerApp}"."${ServerName}.config.conf

export TafnodeLauncherFile=${_TARSNODE_EXECUTION_FILE_}
export TafnodeConfFile=${_TARSNODE_CONF_FILE_}
export TimezoneLauncherFile=${_TARSNODE_UTIL_TIMEZONE_}

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
  ServerLauncherType

  TafnodeLauncherFile
  TafnodeConfFile

  TimezoneLauncherFile
)

echo -e "#! /usr/bin/env bash" >${_TARSNODE_ENVIRONMENT_}
for KEY in "${ForwardENVKeyList[@]}"; do
  echo -e "export ${KEY}=${!KEY}" >>${_TARSNODE_ENVIRONMENT_}
done
