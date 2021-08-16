#! /bin/bash

mkdir -p /buildDir
mkdir -p /uploadDir

TARSIMAGE_EXECUTION_FILE="/usr/local/app/tars/tarsimage/bin/tarsimage"
exec ${TARSIMAGE_EXECUTION_FILE}
