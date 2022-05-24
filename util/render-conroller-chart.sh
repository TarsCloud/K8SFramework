#!/usr/bin/env bash

# shellcheck disable=SC2034
. ./util/common.sh
set -o errexit
set -ex

_WORK_DIR_=$1
_CHART_VERSION_=$2
_CHART_APPVERSION_=$3
_CRD_SERVED_VERSIONS_=$4
read -ra _CRD_SERVED_VERSIONS_ <<<"${_CRD_SERVED_VERSIONS_}"
_CRD_STORAGE_VERSION_=$5
_CONTROLLER_REGISTRY_=$6
_CONTROLLER_TAG_=$7

STORAGE_PLACEHOLDER=_$(echo "${_CRD_STORAGE_VERSION_}" | tr "[:lower:]" "[:upper:]")_STORAGE_
for CRD_FILE in "${_WORK_DIR_}"/crds/*.yaml; do
  ${SED_ALIAS} "s#${STORAGE_PLACEHOLDER}#true#g" "${CRD_FILE}"

  if ! grep "storage: true" "${CRD_FILE}"; then
    for SERVED_VERSION in "${_CRD_SERVED_VERSIONS_[@]}"; do
      BENCH_STORAGE_PLACEHOLDER=_$(echo "${SERVED_VERSION}" | tr "[:lower:]" "[:upper:]")_STORAGE_
      ${SED_ALIAS} "s#${BENCH_STORAGE_PLACEHOLDER}#true#g" "${CRD_FILE}"
      if grep "storage: true" "${CRD_FILE}"; then
        break
      fi
    done
  fi

  ${SED_ALIAS} "s#_.*_STORAGE_#false#g" "${CRD_FILE}"

  for SERVED_VERSION in "${_CRD_SERVED_VERSIONS_[@]}"; do
    SERVED_PLACEHOLDER=_$(echo "${SERVED_VERSION}" | tr "[:lower:]" "[:upper:]")_SERVED_
    ${SED_ALIAS} "s#${SERVED_PLACEHOLDER}#true#g" "${CRD_FILE}"
  done

  ${SED_ALIAS} "s#_.*_SERVED_#false#g" "${CRD_FILE}"

  if ! grep "served: true" "${CRD_FILE}"; then
    rm -rf "${CRD_FILE}"
  fi
done

${SED_ALIAS} "s#version:.*\$#version: ${_CHART_VERSION_}#g" "${_WORK_DIR_}/Chart.yaml"
${SED_ALIAS} "s#appVersion:.*\$#appVersion: ${_CHART_APPVERSION_}#g" "${_WORK_DIR_}/Chart.yaml"

declare -a KeyList=(
  _CONTROLLER_REGISTRY_
  _CONTROLLER_TAG_
)

for KEY in "${KeyList[@]}"; do
  if [ -z "${!KEY}" ]; then
    ${SED_ALIAS} "s#${KEY}#\"\"#g" "${_WORK_DIR_}/values.yaml"
  else
    ${SED_ALIAS} "s#${KEY}#${!KEY}#g" "${_WORK_DIR_}/values.yaml"
  fi
done
