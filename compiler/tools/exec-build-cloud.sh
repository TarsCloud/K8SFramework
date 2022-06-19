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

if [ ! -f $VALUES ] ; then
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
LOGO="`node /root/yaml-tools/index -f $VALUES -n -g cloud.logo`"
CHANGELIST="`node /root/yaml-tools/index -f $VALUES -n -g cloud.changelist`"
TARS="`node /root/yaml-tools/index -f $VALUES -n -g cloud.protocols`"
README="`node /root/yaml-tools/index -f $VALUES -n -g cloud.readme`"
README_CN="`node /root/yaml-tools/index -f $VALUES -n -g cloud.readme_cn`"
ASSETS="`node /root/yaml-tools/index -f $VALUES -n -g cloud.assets`"

if [ -z $GROUP ]; then
    echo "group in ${VALUES} must not be empty"
    exit -1
fi

if [ -z $NAME ]; then
    echo "name in ${VALUES} must not be empty"
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

IMAGE="docker.tarsyun.com/${GROUP}/${NAME}:${TAG}"

# update values image
node /root/yaml-tools/index -f $VALUES -s repo.image -v $IMAGE -u
node /root/yaml-tools/index -f $VALUES -s cloud.version -v $TAG -u
node /root/yaml-tools/index -f $VALUES -s cloud.deploy -v $VALUES -u
node /root/yaml-tools/index -f $VALUES -s cloud.group -v $GROUP -u
node /root/yaml-tools/index -f $VALUES -s cloud.name -v $NAME -u

echo "---------------------Environment---------------------------------"
echo "BIN:                  "$BIN
echo "VALUES:               "$VALUES
echo "BASEIMAGE:            "$BASEIMAGE
echo "SERVERTYPE:           "$SERVERTYPE
echo "PUSH:                 "$PUSH
echo "TAG:                  "$TAG
echo "GROUP:                "$GROUP
echo "NAME:                 "$NAME
echo "LOGO:                 "$LOGO
echo "IMAGE:                "$IMAGE
echo "TARS:                 "$TARS
echo "README:               "$README
echo "README_CN:            "$README_CN
echo "ASSETS:               "$ASSETS
echo "CHANGELIST:           "$CHANGELIST
echo "----------------------Build docker--------------------------------"

NewDockerfile=${Dockerfiile}.new

cp -rf ${Dockerfile} ${NewDockerfile}

echo "COPY $VALUES /usr/local/cloud/cloud.yaml" >> ${NewDockerfile}

for KEY in ${TARS}; do
    echo "COPY $KEY /usr/local/cloud/data/$KEY" >> ${NewDockerfile}
done

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

if [ "$SERVERTYPE" == "nodejs" ]; then
    mkdir -p tars_nodejs
    npm install @tars/node-agent -g
    mv /usr/local/lib/node_modules/@tars/node-agent tars_nodejs/
if

# echo "RUN cd /usr/local/server/bin && tar czfv ${GROUP}.${NAME}.tgz bin && mv ${GROUP}.${NAME}.tgz /usr/local/cloud/data/" >> ${NewDockerfile}

echo "docker build . -f ${NewDockerfile} -t $IMAGE --build-arg BIN=$BIN --build-arg BaseImage=$BASEIMAGE --build-arg ServerType=$SERVERTYPE"
docker build . -f ${NewDockerfile} -t $IMAGE --build-arg BIN=$BIN --build-arg BaseImage=$BASEIMAGE --build-arg ServerType=$SERVERTYPE

rm -rf ${NewDockerfile}

if [ "${PUSH}" == "true" ]; then
    docker push $IMAGE
fi


