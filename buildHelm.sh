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
  LOG_INFO "Usage: $0  <DOCKER_REPOSITORY>  <DOCKER_REGISTRY_USER> <DOCKER_REGISTRY_PASSWORD> <id> <DOCKER_REGISTRY> "
  LOG_INFO "Example: $0 tarscloud user password v2.1 "
  LOG_INFO "Example: $0 tarscloud user password v2.1 harbor.xxx.com "
  exit 255
fi

#docker registry info
_DOCKER_REPOSITORY_=$1
_DOCKER_REGISTRY_USER_=$2
_DOCKER_REGISTRY_PASSWORD_=$3
_BUILD_ID_=$4
_DOCKER_REGISTRY_=$5

echo "---------------------Environment---------------------------------"
echo "DOCKER_REPOSITORY:      "$_DOCKER_REPOSITORY_
echo "DOCKER_REGISTRY_USER:   "$_DOCKER_REGISTRY_USER_
echo "BUILD_ID:               "$_BUILD_ID_
echo "DOCKER_REGISTRY:        "$_DOCKER_REGISTRY_

#
# export DOCKER_CLI_EXPERIMENTAL=enabled 
# docker buildx create --use --name tars-builder-k8s-framework
# docker buildx inspect tars-builder-k8s-framework --bootstrap
# docker run --rm --privileged docker/binfmt:a7996909642ee92942dcd6cff44b9b95f08dad64


#### 构建基础镜像
declare -a BaseImages=(
  tars.cppbase
  tars.javabase
  tars.nodejsbase
  tars.php74base
  tars.tarsnode
  helm.wait
)

for KEY in "${BaseImages[@]}"; do
  # if ! docker buildx build --platform=linux/amd64,linux/arm64 -o type=docker  -t "${KEY}" -f build/"${KEY}.Dockerfile" build; then
  if ! docker build -t "${_DOCKER_REPOSITORY_}/${KEY}" -f build/"${KEY}.Dockerfile" build; then
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
  if ! docker build -t "${_DOCKER_REPOSITORY_}/${KEY}" -f build/"${KEY}.Dockerfile" build; then
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

  echo "FROM ${_DOCKER_REPOSITORY_}/tars.cppbase
ENV ServerType=cpp
COPY /root /
" >build/files/template/tars."${KEY}"/Dockerfile


  # if ! docker buildx build --platform=linux/amd64,linux/arm64 -o type=docker -t tars."${KEY}" build/files/template/tars."${KEY}"; then
  if ! docker build -t ${_DOCKER_REPOSITORY_}/tars."${KEY}" build/files/template/tars."${KEY}"; then
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
  tars.nodejsbase
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
if ! docker login -u "${_DOCKER_REGISTRY_USER_}" -p "${_DOCKER_REGISTRY_PASSWORD_}" "${_DOCKER_REGISTRY_}"; then
  LOG_ERROR "docker login to ${_DOCKER_REGISTRY_} failed!"
  exit 255
fi

for KEY in "${LocalImages[@]}"; do

  # Specified BuildID Tag
  RemoteImagesTag="${_DOCKER_REGISTRY_}/${_DOCKER_REPOSITORY_}/${KEY}":${_BUILD_ID_}
  if ! docker tag "${KEY}" "${RemoteImagesTag}"; then
    LOG_ERROR "Tag ${KEY} image failed"
    exit 255
  fi
  if ! docker push "${RemoteImagesTag}"; then
    LOG_ERROR "Push ${RemoteImagesTag} image failed"
    exit 255
  fi

  # # Latest Tag
  # RemoteImagesLatestTag="${_DOCKER_REGISTRY_}/${_DOCKER_REPOSITORY_}/${KEY}":latest
  # if ! docker tag "${KEY}" "${RemoteImagesLatestTag}"; then
  #   LOG_ERROR "Tag ${KEY} image failed"
  #   exit 255
  # fi
  # if ! docker push "${RemoteImagesLatestTag}"; then
  #   LOG_ERROR "Push ${RemoteImagesLatestTag} image failed"
  #   exit 255
  # fi
done

echo -e "helm:
    build:
      id: ${_BUILD_ID_}
    dockerhub:
      registry: ${_DOCKER_REGISTRY_}
" >install/tarscontroller/values.yaml
helm package install/tarscontroller --version ${_BUILD_ID_} -d ./charts

echo -e "helm:
    build:
      id: ${_BUILD_ID_}
    dockerhub:
      registry: ${_DOCKER_REGISTRY_}
" >./install//values.yaml
helm package install/tarsframework --version ${_BUILD_ID_} -d ./charts

LOG_INFO "Build Helm File OK "
