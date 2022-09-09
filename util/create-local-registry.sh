#!/usr/bin/env bash

# shellcheck disable=SC2034
. ./util/common.sh
set -o errexit
set -ex

# create registry container unless it already exists
reg_name=$1
reg_port=$2
mount_dir=$3

function mount() {
  if [ -n "$1" ]; then
    MountOption="-v $1:/var/lib/registry"
  fi
}

if [ "$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)" != 'true' ]; then
  mount "$mount_dir"
  docker run -d --restart=always -p "127.0.0.1:${reg_port}:5000" --name "${reg_name}" ${MountOption} registry:2
fi
