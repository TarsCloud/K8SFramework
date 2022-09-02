# docker build . -f cpp-compiler-centos.Dockerfile -t jaminzou/cpp-compiler-centos --build-arg BRANCH=master

FROM docker:19.03 AS idocker
RUN mv $(command -v  docker) /tmp/docker

FROM devth/helm:v3.7.1 AS ihelm
RUN mv $(command -v  helm) /tmp/helm

FROM bitnami/kubectl:1.20 AS ikubectl
RUN cp -rf $(command -v  kubectl) /tmp/kubectl


FROM centos:centos7.9.2009

ARG TARGETARCH
RUN echo ${TARGETARCH}

ENV VERSION="v16.17.0"

ARG BRANCH

COPY --from=idocker /tmp/docker /usr/local/bin/docker
COPY --from=ihelm /tmp/helm /usr/local/bin/helm
COPY --from=ikubectl /tmp/kubectl /usr/local/bin/kubectl

RUN yum update -y

RUN yum install -y git make maven gdb bison flex                              \
    ca-certificates openssl telnet curl wget default-mysql-client                      \
    iputils-ping vim tcpdump net-tools binutils procps tree python python3             \
    libssl-dev zlib1g-dev libzip-dev  tzdata localepurge

# centos7的cmake版本太低，需要移除后重新安装
RUN yum install -y gcc gcc-c++ openssl-devel

RUN if [ "${TARGETARCH}" == "amd64" ]; then SUFFIX="x64"; else SUFFIX="arm64"; fi \
    && echo ${VERSION} \
    && echo ${SUFFIX} \
    && cd /root && curl -O https://nodejs.org/dist/${VERSION}/node-${VERSION}-linux-${SUFFIX}.tar.xz \
    && tar -xf node-${VERSION}-linux-${SUFFIX}.tar.xz \
    && ln -s /root/node-${VERSION}-linux-${SUFFIX}/bin/node /usr/bin/node \
    && ln -s /root/node-${VERSION}-linux-${SUFFIX}/bin/npm /usr/bin/npm \
    && ls -l /usr/bin/node && ls -l /usr/bin/npm \
    && /usr/bin/node --version && /usr/bin/npm --version

RUN npm install -g @tars/deploy

RUN yum remove -y cmake                   \
    && mkdir /opt/cmake && cd /opt/cmake/ \
    && wget https://cmake.org/files/v3.16/cmake-3.16.6.tar.gz && tar -zxvf cmake-3.16.6.tar.gz
RUN cd /opt/cmake/cmake-3.16.6 && ./configure --prefix=/usr/local/cmake
RUN cd /opt/cmake/cmake-3.16.6 && make -j4 && make install \
    && ln -s /usr/local/cmake/bin/cmake /usr/bin/cmake && cmake -version

# 编译安装tarscpp
RUN cd /root                                                               \
    && git clone https://github.com/TarsCloud/TarsCpp.git --recursive

RUN cd /root/TarsCpp                                                       \
    && git checkout $BRANCH && git submodule update --remote --recursive

RUN cd /root/TarsCpp && mkdir build && cd build                            \
    && cmake .. && make -j4 && make install                                \
    && cd /                                                                \
    && rm -rf /root/TarsCpp

# 静态编译安装静态库
RUN yum install -y glibc-static libstdc++-static

# deploy tools

COPY tools/yaml-tools /root/yaml-tools
COPY tools/helm-lib /root/helm-lib
COPY tools/helm-template /root/helm-template
COPY tools/Dockerfile /root/Dockerfile

COPY tools/exec-build-cloud.sh /usr/bin/
COPY tools/exec-build-cloud-product.sh /usr/bin/
COPY tools/exec-deploy.sh /usr/bin/
COPY tools/exec-build.sh /usr/bin/
COPY tools/exec-helm.sh /usr/bin/
COPY tools/create-buildx-dockerfile.sh /usr/bin/
COPY tools/create-buildx-dockerfile-product.sh /usr/bin/

RUN chmod a+x /usr/bin/exec-build-cloud.sh
RUN chmod a+x /usr/bin/exec-build-cloud-product.sh
RUN chmod a+x /usr/bin/exec-deploy.sh
RUN chmod a+x /usr/bin/exec-build.sh
RUN chmod a+x /usr/bin/exec-helm.sh
RUN chmod a+x /usr/bin/create-buildx-dockerfile.sh
RUN chmod a+x /usr/bin/create-buildx-dockerfile-product.sh

RUN cd /root/yaml-tools && npm install 

RUN echo "#!/bin/bash" > /bin/start.sh && echo "while true; do sleep 10; done" >> /bin/start.sh && chmod a+x /bin/start.sh

CMD ["/bin/start.sh"]