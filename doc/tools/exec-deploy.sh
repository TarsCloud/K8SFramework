#!/bin/bash
if [ $# -lt 2 ]; then
    echo "Usage: $0 Namespace HelmPackage"
    echo "for example, $0 tars-dev od-storageserver.tgz"
    exit -1
fi

NAMESPACE=$1
HELMPACKAGE=$2
HELMPACKAGE_NAME=$(basename ${HELMPACKAGE})
APP=`echo ${HELMPACKAGE_NAME} | cut -d'-' -f1`
SERVER=`echo ${HELMPACKAGE_NAME} | cut -d'-' -f2`

DATE=`date +"%Y%m%d%H%M%S"`

REPO_ID="v-${DATE}"

echo "---------------------Environment---------------------------------"
echo "DATE:                 "$DATE
echo "NAMESPACE:            "$NAMESPACE
echo "HELMPACKAGE:          "$HELMPACKAGE
echo "HELMPACKAGE_NAME:     "$HELMPACKAGE_NAME
echo "APP:                  "$APP
echo "SERVER:               "$SERVER
echo "REPO_ID:              "$REPO_ID

if [[ "${APP}" == "" ]] || [[ "${SERVER}" == "" ]]; then
    echo "app or server is empty, HELMPACKAGE(${HELMPACKAGE}) is invalid, format must be: app-server.tgz, for example: od-storageserver.tgz"
fi

K8SSERVER="${APP}-${SERVER}"

echo "------------------------deploy------------------------------"

HELM_SERVER=`helm list -f ^${K8SSERVER}$ -n ${NAMESPACE} --deployed -q`
echo "helm list -f ^${K8SSERVER}$ -n ${NAMESPACE} --deployed -q"

echo "HELM_SERVER:$HELM_SERVER, K8SSERVER:$K8SSERVER"

if [ "${HELM_SERVER}" = "${K8SSERVER}" ]; then
    echo "helm upgrade ${K8SSERVER} -n ${NAMESPACE} --set repo.id=${REPO_ID} ${HELMPACKAGE}"
    helm upgrade ${K8SSERVER} -n ${NAMESPACE} --set repo.id=${REPO_ID} ${HELMPACKAGE}

else
    echo "helm install ${K8SSERVER} -n ${NAMESPACE} --set repo.id=${REPO_ID} ${HELMPACKAGE}"
    helm install ${K8SSERVER} -n ${NAMESPACE} --set repo.id=${REPO_ID} ${HELMPACKAGE}
fi

