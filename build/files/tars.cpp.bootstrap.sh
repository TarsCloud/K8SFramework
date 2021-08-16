#!/usr/bin/env bash

echo "-----------------------begin make tars----------------------"

mkdir -p build_tarscpp_release
cd build_tarscpp_release
cmake /tars-k8s-src/TarsCpp -DCMAKE_BUILD_TYPE=release -DTARS_SSL=ON -DONLY_LIB=ON
make -j7; make install
cd ..

echo "-----------------------make tars success----------------------"
