#!/bin/bash

if [ $# -lt 6 ]; then
    echo "Usage: $0 BaseImage SERVERTYPE(cpp/nodejs/java-war/java-jar/go/php) Files YamlFile Tag Push Dockerfile"
    echo "for example, $0 tarscloud/tars.cppbase:v1.0.0 nodejs . yaml/values.yaml latest true Dockerfile"
    echo "for example, $0 tarscloud/tars.cppbase:v1.0.0 cpp build/bin/TestServer yaml/values.yaml v1.0.0 true"
    exit -1
fi

BASEIMAGE=$1
SERVERTYPE=$2
BIN=$3
VALUES=$4
TAG=$5
PUSH=$6
Dockerfile=$7

if [ "$SERVERTYPE" != "cpp" ] && [ "$SERVERTYPE" != "nodejs" ] && [ "$SERVERTYPE" != "java-war" ] && [ "$SERVERTYPE" != "java-jar" ] && [ "$SERVERTYPE" != "go" ] && [ "$SERVERTYPE" != "php" ] ; then  
    echo "Usage: $0 SERVERTYPE(cpp/nodejs/java-war/java-jar/go/php)"
    exit -1
fi

if [ "${PUSH}" == "" ]; then
    PUSH="true"
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

if [ -z $TAG ]; then
    echo "TAG must not be empty, exit"
    exit -1
fi

APP=`node /root/yaml-tools/index -f $VALUES -g app`
SERVER=`node /root/yaml-tools/index -f $VALUES -g server`
IMAGE=`node /root/yaml-tools/index -f $VALUES -g repo.image`
HELM_VERSION=`node /root/yaml-tools/index -f /root/helm-template/Chart.yaml -g version`

if [ -z $IMAGE ]; then
    echo "repo.image in ${VALUES} must not be empty"
    exit -1
fi

IMAGE="$IMAGE:$TAG"

echo "---------------------Environment---------------------------------"
echo "BIN:                  "$BIN
echo "VALUES:               "$VALUES
echo "BASEIMAGE:            "$BASEIMAGE
echo "SERVERTYPE:           "$SERVERTYPE
echo "REGISTRY:             "$REGISTRY
echo "TAG:                  "$TAG
echo "APP:                  "$APP
echo "SERVER:               "$SERVER
echo "PUSH:                 "$PUSH
echo "IMAGE:                "$IMAGE
echo "HELM_VERSION:         "$HELM_VERSION
echo "----------------------Build docker--------------------------------"

echo "docker build . -f ${Dockerfile} -t $IMAGE --build-arg BIN=$BIN --build-arg BaseImage=$BASEIMAGE --build-arg ServerType=$SERVERTYPE"
docker build . -f ${Dockerfile} -t $IMAGE --build-arg BIN=$BIN --build-arg BaseImage=$BASEIMAGE --build-arg ServerType=$SERVERTYPE

if [ "${PUSH}" == "true" ]; then
    docker push $IMAGE
fi


