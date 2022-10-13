#! /bin/bash

.. /bin/timezone.sh

EXECUTION_FILE="/usr/local/app/tars/tarsagent/bin/tarsagent"
chmod +x ${EXECUTION_FILE}
exec ${EXECUTION_FILE}
