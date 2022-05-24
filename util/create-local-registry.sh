#!/usr/bin/env bash

# shellcheck disable=SC2034
. ./util/common.sh
set -o errexit
set -ex

# create registry container unless it already exists
reg_name=$1
reg_port=$2
if [ "$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)" != 'true' ]; then
  docker run -d --restart=always -p "127.0.0.1:${reg_port}:5000" --name "${reg_name}" registry:2
fi
