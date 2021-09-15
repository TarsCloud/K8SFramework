FROM ubuntu:20.04

WORKDIR /root/

ARG BRANCH

ENV GOPATH=/usr/local/go
ENV DEBIAN_FRONTEND=noninteractive
ENV SWOOLE_VERSION=v4.4.16 

# Install
RUN apt update 

RUN apt install -y mysql-client git build-essential unzip make golang flex bison \
    && apt install -y libprotobuf-dev libprotobuf-c-dev zlib1g-dev libssl-dev \
    && apt install -y curl wget net-tools iproute2 \
    #intall php tars
    && apt install -y php php-dev php-cli php-gd php-curl php-mysql \
    && apt install -y php-zip php-fileinfo php-redis php-mbstring tzdata git make wget \
    && apt install -y build-essential libmcrypt-dev php-pear composer\
    # Get and install nodejs
    && apt install -y nodejs npm \ 
    && npm install -g npm pm2 \
    # Get and install JDK
    && apt install -y openjdk-8-jdk maven \
    && apt clean \
    && rm -rf /var/cache/apt


# 安装 kubectl helm 部署工具

RUN curl -s https://mirrors.aliyun.com/kubernetes/apt/doc/apt-key.gpg | apt-key add - 
RUN echo "deb https://mirrors.aliyun.com/kubernetes/apt kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list
RUN apt-get update \
    && apt-get install -y kubectl=1.19.8-00 \
    && apt-mark hold kubectl
RUN curl -O https://tars-thirdpart-1300910346.cos.ap-guangzhou.myqcloud.com/src/helm-v3.5.2-linux-amd64.tar.gz && tar xzf helm-v3.5.2-linux-amd64.tar.gz && mv linux-amd64/helm /usr/local/bin/ && rm helm-v3.5.2-linux-amd64.tar.gz
RUN helm plugin install https://github.com/chartmuseum/helm-push

# 安装docker
RUN apt install -y apt-transport-https ca-certificates curl gnupg2 software-properties-common
RUN curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | apt-key add -
RUN add-apt-repository \
    "deb [arch=amd64] https://mirrors.aliyun.com/docker-ce/linux/ubuntu \
    $(lsb_release -cs) \
    stable"
RUN apt update
RUN apt install -y docker-ce

RUN npm install -g @tars/deploy

# Install cmake for cpp
RUN mkdir -p /tmp/cmake/  \
    && cd /tmp/cmake \
    && curl -O https://tars-thirdpart-1300910346.cos.ap-guangzhou.myqcloud.com/src/cmake-3.19.7.tar.gz  \
    && tar xzf cmake-3.19.7.tar.gz \
    && cd cmake-3.19.7 \
    && ./configure  \
    && make -j4 && make install \
    && rm -rf /tmp/cmake

# Clone Tars repo and init php submodule
RUN cd /root/ && git clone https://github.com/TarsCloud/Tars.git \
    && cd /root/Tars/ \
    && git submodule update --init --recursive php \
    #intall PHP Tars module
    && cd /root/Tars/php/tars-extension/ && phpize \
    && ./configure --enable-phptars && make && make install \
    && echo "extension=phptars.so" > /etc/php/7.4/cli/conf.d/10-phptars.ini \
    # Install PHP swoole module
    && cd /root && git clone https://github.com/swoole/swoole \
    && cd /root/swoole && git checkout $SWOOLE_VERSION \
    && cd /root/swoole \
    && phpize && ./configure --with-php-config=/usr/bin/php-config \
    && make \
    && make install \
    && echo "extension=swoole.so" > /etc/php/7.4/cli/conf.d/20-swoole.ini \
    # Do somethine clean
    && cd /root && rm -rf swoole \
    && mkdir -p /root/phptars && cp -f /root/Tars/php/tars2php/src/tars2php.php /root/phptars 

# Install tars go
RUN go get github.com/TarsCloud/TarsGo/tars \
    && cd $GOPATH/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go \
    && go build .  \
    && mkdir -p /usr/local/go/bin \
    && chmod a+x /usr/local/go/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go/tars2go \
    && ln -s /usr/local/go/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go/tars2go /usr/local/go/bin/tars2go 

RUN cd /root/ \
    && git clone https://github.com/TarsCloud/TarsCpp.git --recursive \ 
    && cd /root/TarsCpp \
    && git checkout $BRANCH && git submodule update --remote --recursive \
    && mkdir -p build \
    && cd build \
    && cmake .. \
    && make -j4 \
    && make install \
    && cd /root \
    && rm -rf /root/TarsCpp

RUN  apt-get clean

COPY tools/yaml-tools /root/yaml-tools
COPY tools/helm-lib /root/helm-lib
COPY tools/helm-template /root/helm-template
COPY tools/Dockerfile /root/Dockerfile
COPY tools/exec-build.sh /usr/bin/
COPY tools/exec-deploy.sh /usr/bin/

RUN cd /root/yaml-tools && npm install 
RUN chmod a+x /usr/bin/exec-deploy.sh
RUN chmod a+x /usr/bin/exec-build.sh

RUN echo "#!/bin/bash" > /bin/start.sh && echo "while true; do sleep 10; done" >> /bin/start.sh

ENTRYPOINT ["/bin/start.sh"]