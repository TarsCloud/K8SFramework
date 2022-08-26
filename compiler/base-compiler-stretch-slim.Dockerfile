# docker build . -f base-compiler-stretch.Dockerfile -t tarscloud/base-compiler-stretch:master --build-arg BRANCH=master

# only support tarscpp

FROM node:lts-stretch AS inode

FROM devth/helm:v3.7.1 AS ihelm
RUN mv $(command -v  helm) /tmp/helm

FROM bitnami/kubectl:1.20 AS ikubectl
RUN cp -rf $(command -v  kubectl) /tmp/kubectl

FROM docker:19.03 AS idocker
RUN mv $(command -v  docker) /tmp/docker

COPY --from=inode /usr/local /usr/local
COPY --from=idocker /tmp/docker /usr/local/bin/docker
COPY --from=ihelm /tmp/helm /usr/local/bin/helm
COPY --from=ikubectl /tmp/kubectl /usr/local/bin/kubectl

ENV PATH=/usr/local/openjdk-8/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

ARG BRANCH
ARG TARS_SSL
# image debian:stretch had "ls bug", we use busybox ls instead

RUN rm -rf /bin/ls

RUN apt update                                                                         \
    && apt install -y git cmake make maven gdb bison flex                              \
    ca-certificates openssl telnet curl wget default-mysql-client                      \
    iputils-ping vim tcpdump net-tools binutils procps tree python python3             \
    libssl-dev zlib1g-dev libzip-dev  tzdata localepurge                               \
    busybox && busybox --install

RUN locale-gen en_US.utf8
ENV LANG en_US.utf8

RUN cd /root                                                                           \
    && git clone https://github.com/TarsCloud/TarsCpp.git --recursive                  \
    && cd /root/TarsCpp                                                                \
    && git checkout $BRANCH && git submodule update --remote --recursive               \
    && mkdir -p build                                                                  \
    && cd build                                                                        \
    && cmake .. -DTARS_SSL=$TARS_SSL                                                   \
    && make -j4                                                                        \
    && make install                                                                    \
    && cd /                                                                            \
    && rm -rf /root/TarsCpp

RUN apt purge -y                                                                       \
    && apt clean all                                                                   \
    && rm -rf /var/lib/apt/lists/*                                                     \
    && rm -rf /var/cache/*.dat-old                                                     \
    && rm -rf /var/log/*.log /var/log/*/*.log

RUN npm install -g @tars/deploy

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