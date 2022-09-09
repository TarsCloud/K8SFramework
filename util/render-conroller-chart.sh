#!/usr/bin/env bash

# shellcheck disable=SC2034
. ./util/common.sh
set -o errexit
set -ex

_CHART_DIR_=$1
_CHART_VERSION_=$2
_CHART_APPVERSION_=$3
_CRD_SERVED_VERSIONS_=${4// /,}
_CRD_STORAGE_VERSION_=$5
_CONTROLLER_REGISTRY_=$6
_CONTROLLER_TAG_=$7

${SED_ALIAS} "s#version:.*\$#version: ${_CHART_VERSION_}#g" "${_CHART_DIR_}/Chart.yaml"
${SED_ALIAS} "s#version:.*\$#version: ${_CHART_VERSION_}#g" "${_CHART_DIR_}/charts/crds/Chart.yaml"
${SED_ALIAS} "s#appVersion:.*\$#appVersion: ${_CHART_APPVERSION_}#g" "${_CHART_DIR_}/Chart.yaml"
${SED_ALIAS} "s#appVersion:.*\$#appVersion: ${_CHART_APPVERSION_}#g" "${_CHART_DIR_}/charts/crds/Chart.yaml"

declare -a KeyList=(
  _CONTROLLER_REGISTRY_
  _CONTROLLER_TAG_
  _CRD_STORAGE_VERSION_
  _CRD_SERVED_VERSIONS_
)

for KEY in "${KeyList[@]}"; do
  if [ -z "${!KEY}" ]; then
    ${SED_ALIAS} "s#${KEY}#\"\"#g" "${_CHART_DIR_}/values.yaml"
  else
    ${SED_ALIAS} "s#${KEY}#${!KEY}#g" "${_CHART_DIR_}/values.yaml"
  fi
done
