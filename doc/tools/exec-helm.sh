#!/bin/bash

if [ $# -lt 2 ]; then
    echo "Usage: $0 YamlFile Image"
    echo "for example, $0 yaml/values.yaml tarscloud/base.gatewayserver"
    exit -1
fi

VALUES=$1
IMAGE=$2

#-------------------------------------------------------------------------------------------

if [ ! -f $VALUES ] && [ ! -d $BIN ] ; then
    echo "yaml file($VALUES) not exists, exit."
    exit -1
fi

if [ -z $IMAGE ]; then
    echo "IMAGE must not be empty, exit"
    exit -1
fi

APP=`node /root/yaml-tools/index -f $VALUES -g app`
SERVER=`node /root/yaml-tools/index -f $VALUES -g server`

K8SSERVER="$APP-$SERVER"

TAG=`echo $IMAGE | cut -d':' -f2`
if [ "$TAG" == "" ]; then
    TAG="latest"
fi

DATE=`date +"%Y%m%d%H%M%S"`

REPO_ID="${DATE}-${TAG}"

echo "---------------------Environment---------------------------------"
echo "VALUES:               "$VALUES
echo "DATE:                 "$DATE
echo "TAG:                  "$TAG
echo "APP:                  "$APP
echo "SERVER:               "$SERVER
echo "REPO_ID:              "$REPO_ID
echo "IMAGE:                "$IMAGE
echo "----------------------Build docker--------------------------------"

cp /root/helm-template/Chart.yaml /tmp/Chart.yaml.backup
#-------------------------------------------------------------------------------------------
function build_helm() 
{
    echo "--------------------build helm------------------------"

    cp -rf ${VALUES} /root/helm-template/values.yaml

    # 修改charts里面的参数
    node /root/yaml-tools/index -f /root/helm-template/Chart.yaml -s name -v $K8SSERVER -u
    node /root/yaml-tools/index -f /root/helm-template/Chart.yaml -s appVersion -v "$TAG" -u

    # 更新values
    node /root/yaml-tools/values -f /root/helm-template/values.yaml -d $REPO_ID -i $IMAGE -u

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

