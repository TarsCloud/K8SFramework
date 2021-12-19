#!/bin/bash

# 生成上到云端应用市场的镜像, 根据value.yaml, 
# 自动生成版本号
# 自动生成镜像地址

# Generate an image to the cloud application market according to value yaml,
# Automatically generate version number
# Automatically generate image address

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

GROUP="`node /root/yaml-tools/index -f $VALUES -g cloud.group`"
NAME="`node /root/yaml-tools/index -f $VALUES -g cloud.name`"
TARS="`node /root/yaml-tools/index -f $VALUES -n -g cloud.tars`"
README="`node /root/yaml-tools/index -f $VALUES -n -g cloud.readme`"
ASSETS="`node /root/yaml-tools/index -f $VALUES -n -g cloud.assets`"

if [ -z $GROUP ]; then
    echo "group in ${MARKET} must not be empty"
    exit -1
fi

if [ -z $NAME ]; then
    echo "name in ${MARKET} must not be empty"
    exit -1
fi

IMAGE="docker.tarsyun.com/${GROUP}/${NAME}:${TAG}"

# update values image
node /root/yaml-tools/index -f $VALUES -s repo.image -v $IMAGE -u
node /root/yaml-tools/index -f $VALUES -s cloud.version -v $TAG -u
node /root/yaml-tools/index -f $VALUES -s cloud.deploy -v $VALUES -u

echo "---------------------Environment---------------------------------"
echo "BIN:                  "$BIN
echo "VALUES:               "$VALUES
echo "BASEIMAGE:            "$BASEIMAGE
echo "SERVERTYPE:           "$SERVERTYPE
echo "REGISTRY:             "$REGISTRY
echo "TAG:                  "$TAG
echo "GROUP:                "$GROUP
echo "NAME:                 "$NAME
echo "PUSH:                 "$PUSH
echo "IMAGE:                "$IMAGE
echo "TARS:                 "$TARS
echo "README:               "$README
echo "ASSETS:               "$ASSETS
echo "----------------------Build docker--------------------------------"

NewDockerfile=${Dockerfiile}.new

cp -rf ${Dockerfile} ${NewDockerfile}

echo $VALUES > cloud.yaml
echo "COPY cloud.yaml /usr/local/cloud/" >> ${NewDockerfile}

echo "COPY $VALUES /usr/local/cloud/data/$VALUES" >> ${NewDockerfile}

for KEY in ${TARS}; do
    echo "COPY $KEY /usr/local/cloud/data/$KEY" >> ${NewDockerfile}
done

if [ "$README" != "" ]; then
    echo "COPY $README /usr/local/cloud/data/$README" >> ${NewDockerfile}
fi

for KEY in ${ASSETS}; do
    echo "COPY $KEY /usr/local/cloud/data/$KEY" >> ${NewDockerfile}
done

echo "docker build . -f ${NewDockerfile} -t $IMAGE --build-arg BIN=$BIN --build-arg BaseImage=$BASEIMAGE --build-arg ServerType=$SERVERTYPE"
docker build . -f ${NewDockerfile} -t $IMAGE --build-arg BIN=$BIN --build-arg BaseImage=$BASEIMAGE --build-arg ServerType=$SERVERTYPE

rm -rf ${NewDockerfile}

if [ "${PUSH}" == "true" ]; then
    docker push $IMAGE
fi


