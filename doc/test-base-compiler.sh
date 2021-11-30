#!/bin/bash

trap 'exit' SIGTERM SIGINT

echo "git clone tarsdemo"
SERVERS_PATH=/root/TarsDemo/Servers

#公共函数
function LOG_ERROR()
{
	if (( $# < 1 ))
	then
		echo -e "\033[33m usesage: LOG_ERROR msg \033[0m";
	fi
	
	local msg=$(date +%Y-%m-%d" "%H:%M:%S);

    msg="${msg} $@";

	echo -e "\033[31m $msg \033[0m";	
}

function LOG_WARNING()
{
	if (( $# < 1 ))
	then
		echo -e "\033[33m usesage: LOG_WARNING msg \033[0m";
	fi
	
	local msg=$(date +%Y-%m-%d" "%H:%M:%S);

    msg="${msg} $@";

	echo -e "\033[33m $msg \033[0m";	
}

function LOG_DEBUG()
{
	if (( $# < 1 ))
	then
		LOG_WARNING "Usage: LOG_DEBUG logmsg";
	fi
	
	local msg=$(date +%Y-%m-%d" "%H:%M:%S);

    msg="${msg} $@";

 	echo -e "\033[40;37m $msg \033[0m";	
}

function LOG_INFO()
{
	if (( $# < 1 ))
	then
		LOG_WARNING "Usage: LOG_INFO logmsg";
	fi
	
	local msg=$(date +%Y-%m-%d" "%H:%M:%S);
	
	for p in $@
	do
		msg=${msg}" "${p};
	done
	
	echo -e "\033[32m $msg \033[0m"  	
}


LOG_DEBUG "Building CPP"
# --------------------------------------cpp--------------------------------------
rm -rf ${SERVERS_PATH}/CppServer/build
mkdir -p ${SERVERS_PATH}/CppServer/build
cd ${SERVERS_PATH}/CppServer/build
cmake .. -DTARS_WEB_HOST=${WEB_HOST}
make -j4
make tar
# make upload

#sleep 20000000
# --------------------------------------php--------------------------------------
LOG_DEBUG "Building PHP"
cd ${SERVERS_PATH}/PhpServer/PHPHttp/src
mkdir servant
composer install
cd ../scripts
./tars2php.sh
cd ../src
rm -rf *.tar.gz
composer run-script deploy
cd ${SERVERS_PATH}/PhpServer/PHPTars/src
mkdir servant
composer install
cd ../scripts
./tars2php.sh
cd ../src
rm -rf *.tar.gz
composer run-script deploy

# --------------------------------------golang--------------------------------------
LOG_DEBUG "Building GoLang"
cd ${SERVERS_PATH}/GoServer/GoHttp
go mod vendor
rm -rf *.tgz
make tar
cd ${SERVERS_PATH}/GoServer/GoTars
go mod vendor
rm -rf *.tgz
make tar

# --------------------------------------java--------------------------------------
LOG_DEBUG "Building Java"
cd ${SERVERS_PATH}/JavaServer/JavaHttp
rm -rf ./target
mvn package
cd ${SERVERS_PATH}/JavaServer/JavaTars
rm -rf ./target
mvn package

# --------------------------------------nodejs--------------------------------------
LOG_DEBUG "Building Nodejs"
source /root/.bashrc
source /etc/profile
npm install -g @tars/deploy
cd ${SERVERS_PATH}/NodejsServer/NodejsHttp
rm -rf *.tgz
npm install
tars-deploy NodejsHttp
cd ${SERVERS_PATH}/NodejsServer/NodejsTars
rm -rf *.tgz
npm install
tars-deploy NodejsTars