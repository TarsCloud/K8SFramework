#!/usr/bin/env bash

# shellcheck disable=SC2034
set -ex

case ${OSTYPE} in
linux*) SED_ALIAS="sed -i" ;;
darwin*) SED_ALIAS="sed -i \"\"" ;;
*)
  echo "unknown: OSTYPE value: ${OSTYPE}"
  exit 255
  ;;
esac
