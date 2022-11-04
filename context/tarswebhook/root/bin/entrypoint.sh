#! /bin/bash

. /bin/timezone.sh

EXECUTION_FILE="/usr/local/app/tars/tarswebhook/bin/tarswebhook"

chmod +x ${EXECUTION_FILE}
exec ${EXECUTION_FILE}
