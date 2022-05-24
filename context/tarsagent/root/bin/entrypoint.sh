#! /bin/bash

TARSIMAGE_EXECUTION_FILE="/usr/local/app/tars/tarsagent/bin/tarsagent"
chmod +x ${TARSIMAGE_EXECUTION_FILE}
exec ${TARSIMAGE_EXECUTION_FILE}
