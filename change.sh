#!/bin/bash

function replaceFilePath()
{
    file=$1
    src=$2
    dst=$3
    newfile=`echo $file | sed "s/$src/$dst/g"`

    echo $newfile
    if [ "$newfile" != "$file" ]; then
        if [ -d $newfile ]; then
            mkdir -p $newfile
        else
            mv $file $newfile
        fi
    fi
}

function replaceFileContent()
{
    file=$1
    src=$2
    dst=$3

    if [ ! -d $file ]; then
        # echo $file
        cp $file ${file}.tmp
        sed "s/$src/$dst/g" ${file}.tmp > ${file}
        rm ${file}.tmp
    fi
}

function replaceContent() 
{
    file=$1
    replaceFileContent $file taf tars
    replaceFileContent $file Taf Tars
    replaceFileContent $file TAF TARS
    replaceFileContent $file jce2case tars2case
    replaceFileContent $file JceCurrent TarsCurrent
    replaceFileContent $file JCE TARS
    replaceFileContent $file Jce Tars
    replaceFileContent $file jce tars
    replaceFileContent $file tarsframework 
    replaceFileContent $file JCE2CPP_FLAG TARS_TOOL_FLAG
}

function replace()
{
    path=$1

    if [ -d $path ]; then
        for file in `find $path`
        do
            replaceFilePath $file taf tars    
        done

        for file in `find $path`
        do
            replaceFilePath $file jce tars    
        done

        for file in `find $path`
        do
            replaceContent $file
        done
    elif [ -f $path ]; then

        replaceContent $path
    fi
}

rm -rf build/.cmake
rm -rf build/files/taf-node
rm -rf build/files/taf-web

replace install

exit 0

replace build
replace doc
replace src/CMakeLists.txt
replace src/AgentServer
replace src/ConfigServer
replace src/ControllerServer
replace src/ImageServer
replace src/LibELSClient
replace src/LibK8SClient
replace src/LogServer
replace src/NodeServer
replace src/QueryServer
replace src/NotifyServer
replace src/PropertyServer
replace src/RegistryServer
replace src/StatServer
replace buildBinary.sh
replace buildHelm.sh
replace readme.md

mv src/k8s.taf.io src/k8s.tars.io

replace src/k8s.tars.io/go.mod
replace src/k8s.tars.io/hack/go.mod
replace src/k8s.tars.io/README.md
replace src/k8s.tars.io/api
replace src/k8s.tars.io/generate.sh

cd src/k8s.tars.io
./generate.sh
cd ../..