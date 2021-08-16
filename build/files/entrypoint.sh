#!/usr/bin/env bash

_TARSNODE_ENTRYPOINT_="/usr/local/app/tars/tarsnode/util/start.sh"
chmod +x ${_TARSNODE_ENTRYPOINT_}
exec ${_TARSNODE_ENTRYPOINT_}
