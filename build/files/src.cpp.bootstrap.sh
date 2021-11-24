#!/usr/bin/env bash

echo "-----------------------begin make src----------------------"

mkdir -p build_src_release
cd build_src_release

cmake /tars-k8s-src/src || exit 255
make -j4 || exit 255

cp /usr/local/tars/cpp/tools/tars2case /tars-k8s-binary/tars2case || exit 255
cp bin/tarsregistry /tars-k8s-binary/tarsregistry || exit 255
cp bin/tarslog /tars-k8s-binary/tarslog || exit 255
cp bin/tarsnotify /tars-k8s-binary/tarsnotify || exit 255
cp bin/tarsconfig /tars-k8s-binary/tarsconfig || exit 255
cp bin/tarscontroller /tars-k8s-binary/tarscontroller || exit 255
cp bin/tarsnode /tars-k8s-binary/tarsnode || exit 255
cp bin/tarsagent /tars-k8s-binary/tarsagent || exit 255
cp bin/tarsimage /tars-k8s-binary/tarsimage || exit 255
cp bin/tarsstat /tars-k8s-binary/tarsstat || exit 255
cp bin/tarsproperty /tars-k8s-binary/tarsproperty || exit 255
cp bin/tarsqueryserver /tars-k8s-binary/tarsquerystat || exit 255
cp bin/tarsqueryserver /tars-k8s-binary/tarsqueryproperty || exit 255
cp bin/tarskevent /tars-k8s-binary/tarskevent || exit 255

cd ..

echo "-----------------------make src success----------------------"
