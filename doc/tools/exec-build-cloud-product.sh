#!/bin/bash

# 生成上到云端应用市场的镜像, 根据value.yaml, 
# 自动生成版本号
# 自动生成镜像地址

# Generate an image to the cloud application market according to value yaml,
# Automatically generate version number
# Automatically generate image address

if [ $# -lt 2 ]; then
    echo "Usage: $0 YamlFile Tag"
    echo "for example, $0 yaml/values.yaml latest "
    exit -1
fi

VALUES=$1
TAG=$2

#-------------------------------------------------------------------------------------------

if [ ! -f $VALUES ] ; then
    echo "yaml file($VALUES) not exists, exit."
    exit -1
fi

if [ -z $TAG ]; then
    echo "TAG must not be empty, exit"
    exit -1
fi

NODEJS_SHELL="/root/yaml-tools/index"
NODEJS_SHELL="/Volumes/MyData/centos/K8SFramework/doc/tools/yaml-tools/index"

GROUP="`node $NODEJS_SHELL -f $VALUES -g cloud.group`"
LOGO="`node $NODEJS_SHELL -f $VALUES -n -g cloud.logo`"
CHANGELIST="`node $NODEJS_SHELL -f $VALUES -n -g cloud.changelist`"
README="`node $NODEJS_SHELL -f $VALUES -n -g cloud.readme`"
README_CN="`node $NODEJS_SHELL -f $VALUES -n -g cloud.readme_cn`"
ASSETS="`node $NODEJS_SHELL -f $VALUES -n -g cloud.assets`"
SERVERS="`node $NODEJS_SHELL -f $VALUES -n -g cloud.servers`"
TYPE="`node $NODEJS_SHELL -f $VALUES -n -g cloud.type`"

if [ "$TYPE" != "product" ] ; then
    echo "type must be product, exit."
    exit -1
fi

if [ -z $GROUP ]; then
    echo "group in ${VALUES} must not be empty"
    exit -1
fi

if [ -z $LOGO ]; then
    echo "logo in ${VALUES} must not be empty"
    exit -1
fi

if [ ! -f $LOGO ] ; then
    echo "logo file($LOGO) not exists, exit."
    exit -1
fi

IMAGE="docker.tarsyun.com/${GROUP}/${GROUP}:${TAG}"

# update values image
node $NODEJS_SHELL -f $VALUES -s cloud.image -v $IMAGE -u
node $NODEJS_SHELL -f $VALUES -s cloud.version -v $TAG -u
node $NODEJS_SHELL -f $VALUES -s cloud.group -v $GROUP -u

echo "---------------------Environment---------------------------------"
echo "VALUES:               "$VALUES
echo "TAG:                  "$TAG
echo "GROUP:                "$GROUP
echo "LOGO:                 "$LOGO
echo "IMAGE:                "$IMAGE
echo "SERVERS:              "$SERVERS
echo "README:               "$README
echo "README_CN:            "$README_CN
echo "ASSETS:               "$ASSETS
echo "CHANGELIST:           "$CHANGELIST
echo "----------------------Build docker--------------------------------"

NewDockerfile="Dockerfiile.tmp"

echo "FROM ubuntu:20.04" > ${NewDockerfile}
echo "COPY $VALUES /usr/local/cloud/cloud.yaml" >> ${NewDockerfile}

if [ "$README" != "" ]; then
    if [ ! -f $README ] ; then
        echo "readme file($README) not exists, exit."
        exit -1
    fi

    echo "COPY $README /usr/local/cloud/data/$README" >> ${NewDockerfile}
fi

if [ "$README_CN" != "" ]; then
    if [ ! -f $README_CN ] ; then
        echo "readme file($README_CN) not exists, exit."
        exit -1
    fi

    echo "COPY $README_CN /usr/local/cloud/data/$README_CN" >> ${NewDockerfile}
fi


if [ "$LOGO" != "" ]; then
    if [ ! -f $LOGO ] ; then
        echo "logo file($LOGO) not exists, exit."
        exit -1
    fi

    echo "COPY $LOGO /usr/local/cloud/data/$LOGO" >> ${NewDockerfile}
fi

if [ "$CHANGELIST" != "" ]; then
    if [ ! -f $CHANGELIST ] ; then
        echo "changelist file($CHANGELIST) not exists, exit."
        exit -1
    fi

    echo "COPY $CHANGELIST /usr/local/cloud/data/$CHANGELIST" >> ${NewDockerfile}
fi

for KEY in ${ASSETS}; do
    echo "COPY $KEY /usr/local/cloud/data/$KEY" >> ${NewDockerfile}
done

echo "docker build . -f ${NewDockerfile} -t $IMAGE "
docker build . -f ${NewDockerfile} -t $IMAGE 

rm -rf ${NewDockerfile}

docker push $IMAGE


