#!/bin/bash

if [ $# -lt 8 ]; then
    echo "Usage: $0 BaseImage SERVERTYPE(cpp/nodejs/java-war/java-jar/go/php) Files YamlFile Namespace Registry Tag Dockerfile"
    echo "for example, $0 tarscloud/tars.cppbase nodejs . yaml/values.yaml true tars-dev Dockerfile"
    echo "for example, $0 tarscloud/tars.cppbase cpp build/bin/TestServer yaml/values.yaml false tars-dev"
    exit -1
fi

BASEIMAGE=$1
SERVERTYPE=$2
BIN=$3
VALUES=$4
NAMESPACE=$5
REGISTRY=$6
TAG=$7
Dockerfile=$8

if [ "$SERVERTYPE" != "cpp" ] && [ "$SERVERTYPE" != "nodejs" ] && [ "$SERVERTYPE" != "java-war" ] && [ "$SERVERTYPE" != "java-jar" ] && [ "$SERVERTYPE" != "go" ] && [ "$SERVERTYPE" != "php" ] ; then  
    echo "Usage: $0 SERVERTYPE(cpp/nodejs/java-war/java-jar/go/php)"
    exit -1
fi

if [ "${Dockerfile}" == "" ]; then
    Dockerfile=/root/Dockerfile/Dockerfile
else
    echo "use ${Dockerfile}"
fi

#-------------------------------------------------------------------------------------------

if [ ! -f $VALUES ] && [ ! -d $BIN ] ; then
    echo "yaml file($VALUES) not exists, exit."
    exit -1
fi

if [ ! -f $BIN ] && [ ! -d $BIN ] ; then
    echo "bin file or dir ($BIN) not exists, exit."
    exit -1
fi

if [ -z $NAMESPACE ]; then
    echo "k8s namespace($NAMESPACE) must not be empty, exit"
    exit -1
fi

if [ -z $TAG ]; then
    echo "TAG must not be empty, exit"
    exit -1
fi

APP=`node /root/yaml-tools/index -f $VALUES -g app`
SERVER=`node /root/yaml-tools/index -f $VALUES -g server`

K8SSERVER="$APP-$SERVER"
IMAGE="$REGISTRY/$APP.$SERVER:$TAG"

DATE=`date +"%Y%m%d%H%M%S"`

REPO_ID="${DATE}-${TAG}"

echo "---------------------Environment---------------------------------"
echo "BIN:                  "$BIN
echo "VALUES:               "$VALUES
echo "BASEIMAGE:            "$BASEIMAGE
echo "SERVERTYPE:           "$SERVERTYPE
echo "NAMESPACE:            "$NAMESPACE
echo "REGISTRY:             "$REGISTRY
echo "DATE:                 "$DATE
echo "TAG:                  "$TAG
echo "APP:                  "$APP
echo "SERVER:               "$SERVER
echo "K8SSERVER:            "$K8SSERVER
echo "REPO_ID:              "$REPO_ID
echo "IMAGE:                "$IMAGE
echo "----------------------Build docker--------------------------------"

echo "docker build . -f ${Dockerfile} -t $IMAGE --build-arg BIN=$BIN --build-arg BaseImage=$BASEIMAGE --build-arg ServerType=$SERVERTYPE"
docker build . -f ${Dockerfile} -t $IMAGE --build-arg BIN=$BIN --build-arg BaseImage=$BASEIMAGE --build-arg ServerType=$SERVERTYPE

cp /root/helm-template/Chart.yaml /tmp/Chart.yaml.backup
#-------------------------------------------------------------------------------------------
function build_helm() 
{
    echo "--------------------build helm: ${1} ------------------------"

    cp -rf ${VALUES} /root/helm-template/values.yaml

    # 修改charts里面的参数
    node /root/yaml-tools/index -f /root/helm-template/Chart.yaml -s name -v $K8SSERVER -u
    node /root/yaml-tools/index -f /root/helm-template/Chart.yaml -s appVersion -v "$TAG" -u
    # node /root/yaml-tools/index -f /root/helm-template/Chart.yaml -s dependencies[0].repository -v $TMP_HELM_CHARTS -u

    # 更新values
    node /root/yaml-tools/values -f /root/helm-template/values.yaml -d $REPO_ID -i $IMAGE -u
    node /root/yaml-tools/k8s.js -f /root/helm-template/values.yaml -n ${NAMESPACE} -i $IMAGE -d ${TAG} -u

    helm dependency update /root/helm-template

    helm package /root/helm-template --version "$TAG"
    
    echo "---------------------helm chart--------------------------"
    cat /root/helm-template/Chart.yaml
}


# build helm包
build_helm 

#restore chart.yaml
cp /tmp/Chart.yaml.backup /root/helm-template/Chart.yaml
echo "----------------finish $K8SSERVER---------------------"

