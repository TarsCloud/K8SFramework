#!/usr/bin/env bash

git clone https://github.com/TarsCloud/TarsCpp --recursive
git clone https://github.com/TarsCloud/TarsWeb build/TarsWeb
cd build/TarsWeb
git checkout k8s
