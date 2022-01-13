#!/usr/bin/env bash

_IMAGE_BIND_SERVER_DIR_="/usr/local/server/bin"
_IMAGE_BIND_ENVIRONMENT_FILE_="/usr/local/app/tars/tarsnode/util/environment"

if [ ! -f "${_IMAGE_BIND_ENVIRONMENT_FILE_}" ]; then
  echo "${_IMAGE_BIND_ENVIRONMENT_FILE_} not exist"
  exit 255
fi

source "${_IMAGE_BIND_ENVIRONMENT_FILE_}"

declare -a ExpectENVKeyList=(
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

for KEY in "${ExpectENVKeyList[@]}"; do
  if [ -z "${!KEY}" ]; then
    echo "got empty [${KEY}] env value"
    exit 255
  fi
done

source "${TimezoneLauncherFile}"

echo "${PodIP}" "${PodName}" >>/etc/hosts
echo "${PodIP}" "${ListenAddress}" >>/etc/hosts

mkdir -p "${ServerBaseDir}"
mkdir -p "${ServerConfDir}"
mkdir -p "${ServerDataDir}"

ln -s "${_IMAGE_BIND_SERVER_DIR_}" "${ServerBinDir}"

case ${ServerType} in
"cpp" | "nodejs-pkg" | "go")
  export LD_LIBRARY_PATH=${ServerBinDir}:${ServerBinDir}/lib
  export ServerLauncherFile="${ServerBinDir}/${ServerName}"

  if [ ! -f "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile file not exist"
    exit 255
  fi

  chmod +x "${ServerLauncherFile}"

  if [ ! -x "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile had no execution permission"
    exit 255
  fi
  export ServerLauncherArgv="${ServerName} --config=${ServerConfFile}"
  ;;
"nodejs")
  ServerLauncherFile=$(command -v node)
  export ServerLauncherFile

  if [ ! -f "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile file not exist"
    exit 255
  fi

  chmod +x "${ServerLauncherFile}"

  if [ ! -x "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile had no execution permission"
    exit 255
  fi

  if [ -z "${NODE_AGENT_BIN}" ]; then
    echo "got empty [NODE_AGENT_BIN] env value"
    exit 255
  fi

  chmod +x "${NODE_AGENT_BIN}"

  if [ ! -x "${NODE_AGENT_BIN}" ]; then
    echo "$NODE_AGENT_BIN had no execution permission"
    exit 255
  fi

  export ServerLauncherArgv="node ${NODE_AGENT_BIN} ${ServerBinDir}/ -c ${ServerConfFile}"
  ;;
"java-war")
  ServerLauncherFile=$(command -v java)
  export ServerLauncherFile

  if [ ! -f "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile file not exist"
    exit 255
  fi

  chmod +x "${ServerLauncherFile}"

  if [ ! -x "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile had no execution permission"
    exit 255
  fi
  export ServerLauncherArgv="java -Dconfig=${ServerConfFile} #{jvmparams} -cp #{classpath} #{mainclass}"
  ;;
"java-jar")
  ServerLauncherFile=$(command -v java)
  export ServerLauncherFile

  if [ ! -f "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile file not exist"
    exit 255
  fi

  chmod +x "${ServerLauncherFile}"

  if [ ! -x "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile had no execution permission"
    exit 255
  fi
  export ServerLauncherArgv="java -Dconfig=${ServerConfFile} #{jvmparams} -jar ${ServerBinDir}/${ServerName}.jar"
  ;;
esac

chmod +x "${TafnodeLauncherFile}"
case ${ServerLauncherType} in
foreground)
  OUTFILE=/tmp/argvs
  if ! "${TafnodeLauncherFile}" --config="${TafnodeConfFile}" --target=config --outfile="${OUTFILE}"; then
    echo "generate server template config file error"
    exit 255
  fi

  argv=$(cat ${OUTFILE})

  eval "${TafnodeLauncherFile} --config=${TafnodeConfFile} --target=daemon&"
  eval "exec ${ServerLauncherFile} ${argv}"
  ;;
background)
  eval "exec ${TafnodeLauncherFile} --config=${TafnodeConfFile}"
  ;;
*)
  eval "exec ${TafnodeLauncherFile} --config=${TafnodeConfFile}"
  ;;
esac
