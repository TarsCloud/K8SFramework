# docker build . -f cpp-compiler-centos.Dockerfile -t jaminzou/cpp-compiler-centos --build-arg BRANCH=master

FROM centos:centos7.9.2009

ARG BRANCH

RUN yum update -y

RUN yum install -y git make maven gdb bison flex                              \
    ca-certificates openssl telnet curl wget default-mysql-client                      \
    iputils-ping vim tcpdump net-tools binutils procps tree python python3             \
    libssl-dev zlib1g-dev libzip-dev  tzdata localepurge

# centos7的cmake版本太低，需要移除后重新安装
RUN yum install -y gcc gcc-c++ openssl-devel
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

RUN echo "#!/bin/bash" > /bin/start.sh && echo "while true; do sleep 10; done" >> /bin/start.sh && chmod a+x /bin/start.sh

CMD ["/bin/start.sh"]