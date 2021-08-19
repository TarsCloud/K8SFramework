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

function LOG_DEBUG() {
  msg=$(date +%Y-%m-%d" "%H:%M:%S)
  msg="${msg} $*"
  echo -e "\033[40;37m $msg \033[0m"
}
###### LOG

### build docker for compiler
if ! docker build -t tars.builder -f build/tars.builder.Dockerfile build; then
  LOG_ERROR "Build \"tars.builder\" image error"
  exit 255
  end
fi

### build src
if ! docker run -i -v "${PWD}":/tars-k8s-src -v "${PWD}"/build/files/binary:/tars-k8s-binary tars.builder; then
  LOG_ERROR "Build Source Error"
  exit 255
  end
fi

