#! /bin/bash

TARSIMAGE_EXECUTION_FILE="/usr/local/app/tars/tarsimage/bin/tarsimage"
chmod +x ${TARSIMAGE_EXECUTION_FILE}
exec ${TARSIMAGE_EXECUTION_FILE}
