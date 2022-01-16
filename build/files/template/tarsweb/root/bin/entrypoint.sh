#!/usr/bin/env bash

# TOKEN=$(cat "/var/run/secrets/kubernetes.io/serviceaccount/token")
# NAMESPACE=$(cat "/var/run/secrets/kubernetes.io/serviceaccount/namespace")
# NAME="tars-framework"

# GROUP="k8s.tars.io"
# VERSION="v1beta2"
# KIND="tframeworkconfigs"

# declare -a ExpectENVKeyList=(
#   KUBERNETES_SERVICE_HOST
#   KUBERNETES_SERVICE_PORT
#   TOKEN
#   NAMESPACE
# )

# for KEY in "${ExpectENVKeyList[@]}"; do
#   if [ -z "${!KEY}" ]; then
#     echo "got empty [${KEY}] value"
#     exit 255
#   fi
# done

# CURL_COMMAND="curl -sk -XGET  -H \"Accept: application/json;\" -H \"Authorization: Bearer ${TOKEN}\" \
# \"https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/apis/${GROUP}/${VERSION}/namespaces/${NAMESPACE}/${KIND}/${NAME}\" "

# CONTENT=$(eval "$CURL_COMMAND")

# echo "${CONTENT}" | jq -r .expand.nativeFrameworkConfig >/mnt/config/nativeFramework.conf
# echo "${CONTENT}" | jq -r .expand.nativeDBConfig >/mnt/config/nativeDB.conf

cd /tars-web || exit

# exec pm2 start bin/www --name=tars-node-web --no-daemon
exec npm run prd
