#!/bin/bash

_K8S_POD_IP_=${PodIP}

REGISTRY_EXECUTION_FILE=/usr/local/app/tars/tarsregistry/bin/tarsregistry

REGISTRY_CONFIG_FILE=/usr/local/app/tars/tarsregistry/conf/tarsregistry.conf

declare -a ReplaceKeyList=(
  _K8S_POD_IP_
)

declare -a ReplaceFileList=(
  "${REGISTRY_CONFIG_FILE}"
)

for KEY in "${ReplaceKeyList[@]}"; do
  for FILE in "${ReplaceFileList[@]}"; do
    sed -i "s#${KEY}#${!KEY}#g" "${FILE}"
    if [[ 0 -ne $? ]]; then
      exit 255
    fi
  done
done

ldconfig

exec ${REGISTRY_EXECUTION_FILE} --config=${REGISTRY_CONFIG_FILE}
# exec /bin/guard
