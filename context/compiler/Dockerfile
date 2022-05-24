FROM golang:1.16-bullseye
RUN go env -w GOPROXY=https://goproxy.io,direct

COPY root /

RUN apt update                                                                          \
    && apt install make cmake flex bison libssl-dev zlib1g-dev -y                       \
    && apt purge -y                                                                     \
    && apt clean all                                                                    \
    && rm -rf /var/lib/apt/lists/*                                                      \
    && rm -rf /var/cache/*.dat-old                                                      \
    && rm -rf /var/log/*.log /var/log/*/*.log

ARG DOCKERFILE_MD5
LABEL DOCKERFILE_MD5=$DOCKERFILE_MD5

ARG TARS_CPP_GIT_ADDRESS
ARG TARS_CPP_GIT_BRANCH_OR_TAG
LABEL TARS_CPP_GIT_BRANCH_OR_TAG=$TARS_CPP_GIT_BRANCH_OR_TAG

RUN cd /root/TarsCpp                                                                    \
    && mkdir -p build                                                                   \
    && cd build                                                                         \
    && cmake ..                                                                         \
    && make -j4                                                                         \
    && make install                                                                     \
    && rm -rf /root/Tarscpp

ENTRYPOINT [ "/usr/bin/make","-f","/root/Makefile" ]
