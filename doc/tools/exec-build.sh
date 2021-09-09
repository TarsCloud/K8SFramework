#!/bin/bash

if [ $# -lt 5 ]; then
    echo "Usage: $0 LANG(cpp/nodejs/java-war/java-jar/go/php) Files YamlFile Namespace Registry Tag Dockerfile"
    echo "for example, $0 nodejs . yaml/values.yaml true tars-dev Dockerfile"
    echo "for example, $0 cpp build/bin/TestServer yaml/values.yaml false tars-dev"
    exit -1
fi

LANG=$1

if [ "$LANG" != "cpp" ] && [ "$LANG" != "nodejs" ] && [ "$LANG" != "java-war" ] && [ "$LANG" != "java-jar" ] && [ "$LANG" != "go" ] && [ "$LANG" != "php" ] ; then  
    echo "Usage: $0 LANG(cpp/nodejs/java-war/java-jar/go/php)"
    exit -1
fi

BIN=$2
VALUES=$3
NAMESPACE=$4
REGISTRY=$5
TAG=$6
Dockerfile=$7

if [ "${Dockerfile}" == "" ]; then
    echo "use /root/Dockerfile/Dockerfile.${LANG}"
    Dockerfile=/root/Dockerfile/Dockerfile.${LANG}
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
IMAGE="$REGISTRY/$APP/$SERVER:$TAG"

DATE=`date +"%Y%m%d%H%M%S"`

REPO_ID="${DATE}-${TAG}"

echo "---------------------Environment---------------------------------"
echo "BIN:                  "$BIN
echo "VALUES:               "$VALUES
echo "NAMESPACE:            "$NAMESPACE
echo "REGISTRY:             "$REGISTRY
echo "DATE:                 "$DATE
echo "TAG:                  "$TAG
echo "APP:                  "$APP
echo "SERVER:               "$SERVER
echo "K8SSERVER:            "$K8SSERVER
echo "REPO_ID:              "$REPO_ID
echo "IMAGE:                "$IMAGE
# echo "CHARTS:               "$CHARTS

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

echo "----------------------Build docker--------------------------------"

echo "docker build . -f ${Dockerfile} -t $IMAGE --build-arg BIN=$BIN"
docker build . -f ${Dockerfile} -t $IMAGE --build-arg BIN=$BIN

# build helm包
build_helm 

#restore chart.yaml
cp /tmp/Chart.yaml.backup /root/helm-template/Chart.yaml
echo "----------------finish $K8SSERVER---------------------"

