#!/usr/bin/env bash

###### LOG 函数
function LOG_ERROR() {
  msg=$(date +%Y-%m-%d" "%H:%M:%S)
  msg="${msg} $*"
  echo -e "\033[31m $msg \033[0m"
}

function LOG_WARNING() {
  msg=$(date +%Y-%m-%d" "%H:%M:%S)
  msg="${msg} $*"
  echo -e "\033[33m $msg \033[0m"
}

function LOG_INFO() {
  echo -e "\033[31m $* \033[0m"
}

function LOG_DEBUG() {
  msg=$(date +%Y-%m-%d" "%H:%M:%S)
  msg="${msg} $*"
  echo -e "\033[40;37m $msg \033[0m"
}
###### LOG 

if (($# < 4)); then
  LOG_INFO "Usage: $0 <DOCKER_REGISTRY_URL> <DOCKER_REGISTRY_USER> <DOCKER_REGISTRY_PASSWORD> [id]"
  LOG_INFO "Example: $0 dockerhub.com/tarsk8s tarsk8s tarsk8s@image v2.1"
  exit 255
fi

#docker registry info
_DOCKER_REGISTRY_URL_=$1
_DOCKER_REGISTRY_USER_=$2
_DOCKER_REGISTRY_PASSWORD_=$3
_BUILD_ID_=$4
#


# export DOCKER_CLI_EXPERIMENTAL=enabled 
# docker buildx create --use --name tars-builder-k8s-framework
# docker buildx inspect tars-builder-k8s-framework --bootstrap
# docker run --rm --privileged docker/binfmt:a7996909642ee92942dcd6cff44b9b95f08dad64


#### 构建基础镜像
declare -a BaseImages=(
  tars.cppbase
  tars.javabase
  tars.nodejs10base
  tars.nodejs12base
  tars.nodejs14base
  tars.php74base
  tars.tarsnode
  helm.wait
)

for KEY in "${BaseImages[@]}"; do
  # if ! docker buildx build --platform=linux/amd64,linux/arm64 -o type=docker  -t "${KEY}" -f build/"${KEY}.Dockerfile" build; then
  if ! docker build -t "${KEY}" -f build/"${KEY}.Dockerfile" build; then
    LOG_ERROR "Build ${KEY} image failed"
    exit 255
  fi
done
#### 构建基础镜像

#### 构建基础服务镜像
declare -a FrameworkImages=(
  tarscontroller
  tarsagent
  tars.tarsregistry
  tars.tarsimage
  tars.tarsweb
  tars.elasticsearch
)

for KEY in "${FrameworkImages[@]}"; do
  # if ! docker buildx build --platform=linux/amd64,linux/arm64 -o type=docker -t "${KEY}" -f build/"${KEY}.Dockerfile" build; then
  if ! docker build -t "${KEY}" -f build/"${KEY}.Dockerfile" build; then
    LOG_ERROR "Build ${KEY} image failed"
    exit 255
  fi
done

#### 构建基础服务镜像
declare -a ServerImages=(
  tarslog
  tarsconfig
  tarsnotify
  tarsstat
  tarsproperty
  tarsquerystat
  tarsqueryproperty
)

# #--------------------------------------------------------------------------------------------

for KEY in "${ServerImages[@]}"; do
  mkdir -p build/files/template/tars."${KEY}"
  mkdir -p build/files/template/tars."${KEY}"/root/usr/local/server/bin

  if ! cp build/files/binary/"${KEY}" build/files/template/tars."${KEY}"/root/usr/local/server/bin/"${KEY}"; then
    LOG_ERROR "copy ${KEY} failed, please check ${KEY} is in directory: build/files/binary"
    exit 255
  fi

  echo "FROM tars.cppbase
ENV ServerType=cpp
COPY /root /
" >build/files/template/tars."${KEY}"/Dockerfile


  # if ! docker buildx build --platform=linux/amd64,linux/arm64 -o type=docker -t tars."${KEY}" build/files/template/tars."${KEY}"; then
  if ! docker build -t tars."${KEY}" build/files/template/tars."${KEY}"; then
    LOG_ERROR "Build ${KEY} image failed"
    exit 255
  fi
done

#### 构建基础服务镜像

LOG_INFO "Build All Images Ok"

declare -a LocalImages=(
  tars.tarsnode
  tars.cppbase
  tars.javabase
  tars.nodejs10base
  tars.nodejs12base
  tars.nodejs14base
  tars.php74base
  tarscontroller
  tarsagent
  tars.elasticsearch
  tars.tarsregistry
  tars.tarsimage
  tars.tarsweb
  tars.tarslog
  tars.tarsconfig
  tars.tarsnotify
  tars.tarsstat
  tars.tarsquerystat
  tars.tarsproperty
  tars.tarsqueryproperty
  helm.wait
)

# 登陆
if ! docker login -u "${_DOCKER_REGISTRY_USER_}" -p "${_DOCKER_REGISTRY_PASSWORD_}" "${_DOCKER_REGISTRY_URL_}"; then
  LOG_ERROR "docker login to ${_DOCKER_REGISTRY_URL_} failed!"
  exit 255
fi

for KEY in "${LocalImages[@]}"; do
  # Specified BuildID Tag
  RemoteImagesTag="${_DOCKER_REGISTRY_URL_}"/"${KEY}":${_BUILD_ID_}
  if ! docker tag "${KEY}" "${RemoteImagesTag}"; then
    LOG_ERROR "Tag ${KEY} image failed"
    exit 255
  fi
  if ! docker push "${RemoteImagesTag}"; then
    LOG_ERROR "Push ${RemoteImagesTag} image failed"
    exit 255
  fi
  # Latest Tag
  RemoteImagesLatestTag="${_DOCKER_REGISTRY_URL_}"/"${KEY}":latest
  if ! docker tag "${KEY}" "${RemoteImagesLatestTag}"; then
    LOG_ERROR "Tag ${KEY} image failed"
    exit 255
  fi
  if ! docker push "${RemoteImagesLatestTag}"; then
    LOG_ERROR "Push ${RemoteImagesLatestTag} image failed"
    exit 255
  fi
done

echo -e "helm:
    build:
      id: ${_BUILD_ID_}
    dockerhub:
      registry: ${_DOCKER_REGISTRY_URL_}
" >install/tarscontroller/values.yaml
helm package install/tarscontroller --version ${_BUILD_ID_} -d ./charts

echo -e "helm:
    build:
      id: ${_BUILD_ID_}
    dockerhub:
      registry: ${_DOCKER_REGISTRY_URL_}
" >./install//values.yaml
helm package install/tarsframework --version ${_BUILD_ID_} -d ./charts

LOG_INFO "Build Helm File OK "
