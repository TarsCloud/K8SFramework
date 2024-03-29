# docker build . -f base-compiler-stretch.Dockerfile -t tarscloud/base-compiler-stretch:master --build-arg master
FROM golang:1.17-stretch AS igolang

FROM php:7.3.29-apache-stretch
ENV DEBIAN_FRONTEND=noninteractive

# COPY --from=itars /usr/local /usr/local
COPY --from=igolang /usr/local /usr/local
# COPY --from=iphp /usr/local /usr/local
# COPY --from=ijava /usr/local /usr/local
# COPY --from=inode /usr/local /usr/local
# COPY --from=idocker /tmp/docker /usr/local/bin/docker
# COPY --from=ihelm /tmp/helm /usr/local/bin/helm
# COPY --from=ikubectl /tmp/kubectl /usr/local/bin/kubectl

ENV PATH=/usr/local/openjdk-8/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
ENV GOPATH=/go

# image debian:stretch had "ls bug", we use busybox ls instead

RUN rm -rf /bin/ls

RUN apt update                                                                         \
    && apt install -y git cmake make maven gdb                                         \
    ca-certificates openssl telnet curl wget default-mysql-client                      \
    iputils-ping vim tcpdump net-tools binutils procps tree python python3             \
    libssl-dev zlib1g-dev libzip-dev  tzdata localepurge                               \
    busybox && busybox --install

RUN locale-gen en_US.utf8
ENV LANG en_US.utf8

RUN go get github.com/TarsCloud/TarsGo/tars \
    && cd $GOPATH/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go \
    && go build .  \
    && mkdir -p /usr/local/go/bin \
    && chmod a+x $GOPATH/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go/tars2go \
    && ln -s $GOPATH/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go/tars2go /usr/local/go/bin/tars2go 

# RUN apt update && apt install -y                                                       \
#     g++ make cmake flex bison git ca-certificates curl wget libssl-dev zlib1g-dev

RUN cd /root                                                                           \
    && git clone https://github.com/TarsCloud/TarsCpp.git --recursive                  \
    && cd /root/TarsCpp                                                                \
    && git checkout $BRANCH && git submodule update --remote --recursive               \
    && mkdir -p build                                                                  \
    && cd build                                                                        \
    && cmake ..                                                                        \
    && make -j4                                                                        \
    && make install                                                                    \
    && rm -rf /root/TarsCpp

RUN apt purge -y                                                                       \
    && apt clean all                                                                   \
    && rm -rf /var/lib/apt/lists/*                                                     \
    && rm -rf /var/cache/*.dat-old                                                     \
    && rm -rf /var/log/*.log /var/log/*/*.log

RUN curl -sS https://getcomposer.org/installer | php \
    && mv composer.phar /usr/local/bin/composer \
    && chmod +x /usr/local/bin/composer

RUN npm install -g @tars/deploy

COPY tools/yaml-tools /root/yaml-tools
COPY tools/helm-lib /root/helm-lib
COPY tools/helm-template /root/helm-template
COPY tools/Dockerfile /root/Dockerfile

COPY tools/exec-build.sh /usr/bin/
COPY tools/exec-build-cloud.sh /usr/bin/
COPY tools/exec-build-cloud-product.sh /usr/bin/
COPY tools/exec-deploy.sh /usr/bin/
COPY tools/exec-helm.sh /usr/bin/

RUN cd /root/yaml-tools && npm install 
RUN chmod a+x /usr/bin/exec-deploy.sh
RUN chmod a+x /usr/bin/exec-build.sh
RUN chmod a+x /usr/bin/exec-build-cloud.sh
RUN chmod a+x /usr/bin/exec-helm.sh

COPY test-base-compiler.sh /root/
RUN chmod a+x /root/test-base-compiler.sh

RUN cd /root && git clone https://github.com/TarsCloud/TarsDemo \
    && /root/test-base-compiler.sh \
    && rm -rf /root/TarsDemo

RUN echo "#!/bin/bash" > /bin/start.sh && echo "while true; do sleep 10; done" >> /bin/start.sh && chmod a+x /bin/start.sh

CMD ["/bin/start.sh"]
