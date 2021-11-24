#!/usr/bin/env bash

_IMAGE_BIND_TARSNODE_EXECUTION_FILE_="/usr/local/app/tars/tarsnode/bin/tarsnode"
_IMAGE_BIND_TARSNODE_CONF_FILE_="/usr/local/app/tars/tarsnode/conf/tarsnode.conf"
_TARSNODE_ENVIRONMENT_FILE_="/usr/local/app/tars/tarsnode/util/environment"

_IMAGE_BIND_SERVER_DIR_="/usr/local/server/bin"

source $_TARSNODE_ENVIRONMENT_FILE_

mkdir -p "${ServerBaseDir}"
mkdir -p "${ServerConfDir}"
mkdir -p "${ServerDataDir}"

ln -s "${_IMAGE_BIND_SERVER_DIR_}" "${ServerBinDir}"

if [ -z "${PodIP}" ]; then
  echo "got empty [PodIP] env value"
  exit 255
fi

if [ -z "${PodName}" ]; then
  echo "got empty [PodName] env value"
  exit 255
fi

echo "${PodIP}" "${PodName}" >>/etc/hosts
echo "${PodIP}" "${ListenAddress}" >>/etc/hosts

if [ -z "${ServerType}" ]; then
  echo "got empty [ServerType] env value"
  exit 255
fi

case ${ServerType} in
"cpp" | "go")
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


  export ServerLauncherFile=$(command -v node) 

  if [ ! -f "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile file not exist"
    exit 255
  fi

  chmod +x ${ServerLauncherFile}

  if [ ! -x "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile had no execution permission"
    exit 255
  fi

  if [ -z "${NODE_AGENT_BIN}" ]; then
    echo "got empty [NODE_AGENT_BIN] env value"
    exit 255
  fi

  if [ ! -x "$NODE_AGENT_BIN" ]; then
    echo "$NODE_AGENT_BIN had no execution permission"
    exit 255
  fi

  export ServerLauncherArgv="node $NODE_AGENT_BIN ${ServerBinDir}/ -c ${ServerConfFile}"
  ;;
"java-war")
  export ServerLauncherFile=$JAVA_HOME/bin/java

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
  export ServerLauncherFile=$JAVA_HOME/bin/java

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
"php")
  export ServerLauncherFile=$(command -v php)
  if [ ! -f "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile file not exist"
    exit 255
  fi

  if [ ! -x "$ServerLauncherFile" ]; then
    echo "$ServerLauncherFile had no execution permission"
    exit 255
  fi
  export ServerLauncherArgv="php ${ServerBinDir}/src/index.php --config=${ServerConfFile} "
  ;;
esac

exec ${_IMAGE_BIND_TARSNODE_EXECUTION_FILE_} --config=${_IMAGE_BIND_TARSNODE_CONF_FILE_}
