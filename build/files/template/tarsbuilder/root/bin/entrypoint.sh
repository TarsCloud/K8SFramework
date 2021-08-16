#!/usr/bin/env bash

BUILD_TMP=/tars-k8s-src/build-tmp

mkdir -p ${BUILD_TMP}

# 编译tars静态库依赖
cd ${BUILD_TMP} || exit 255

cp -rf /tars-k8s-src/build/files/src.cpp.bootstrap.sh . || exit 255
cp -rf /tars-k8s-src/build/files/tars.cpp.bootstrap.sh . || exit 255

sh ./tars.cpp.bootstrap.sh || exit 255

sh ./src.cpp.bootstrap.sh || exit 255
