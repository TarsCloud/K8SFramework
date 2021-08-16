#!/usr/bin/env bash

# 安装 tafcontroller
helm install tafcontroller ./install/tafcontroller-*.tgz

# 等待 tafcontroller 启动
sleep 3m

# 安装 tafframework
helm install tafframework install/tafframework-*.tgz -f install/config.yaml