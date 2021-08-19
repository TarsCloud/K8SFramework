#!/usr/bin/env bash

# 安装 tarscontroller
helm install tarscontroller ./install/tarscontroller-*.tgz

# 等待 tarscontroller 启动
sleep 3m

# 安装 tarsframework
helm install tarsframework install/tarsframework-*.tgz -f install/config.yaml