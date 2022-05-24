#!/usr/bin/env bash

# shellcheck disable=SC2034
. ./util/common.sh
set -o errexit
set -ex

_WORK_DIR_=$1
_CHART_VERSION_=$2
_CHART_APPVERSION_=$3
_FRAMEWORK_REGISTRY_=$4
_FRAMEWORK_TAG_=$5

${SED_ALIAS} "s#version:.*\$#version: ${_CHART_VERSION_}#g" "${_WORK_DIR_}/Chart.yaml"
${SED_ALIAS} "s#appVersion:.*\$#appVersion: ${_CHART_APPVERSION_}#g" "${_WORK_DIR_}/Chart.yaml"

declare -a KeyList=(
  _FRAMEWORK_REGISTRY_
  _FRAMEWORK_TAG_
)

for KEY in "${KeyList[@]}"; do
  if [ -z "${!KEY}" ]; then
    ${SED_ALIAS} "s#${KEY}#\"\"#g" "${_WORK_DIR_}/values.yaml"
  else
    ${SED_ALIAS} "s#${KEY}#${!KEY}#g" "${_WORK_DIR_}/values.yaml"
  fi
done
