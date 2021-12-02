FROM golang:1.16-bullseye

# image debian:bullseye had "ls bug", we use busybox ls instead
RUN rm -rf /bin/ls

# /etc/localtime as a solt link will block container mount /etc/localtime from host
RUN rm -rf /etc/localtime

RUN apt update                                                                         \
    && apt install                                                                     \
    make cmake flex bison                                                              \
    ca-certificates openssl telnet curl wget default-mysql-client                      \
    iputils-ping vim tcpdump net-tools binutils procps tree                            \
    libssl-dev zlib1g-dev libprotobuf-dev libprotobuf-c-dev                            \
    busybox -y && busybox --install

RUN apt purge -y                                                                       \
    && apt clean all                                                                   \
    && rm -rf /var/lib/apt/lists/*                                                     \
    && rm -rf /var/cache/*.dat-old                                                     \
    && rm -rf /var/log/*.log /var/log/*/*.log

RUN cd /root                                                                           \
    && git clone https://github.com/TarsCloud/Tars.git                                 \
    && cd /root/Tars                                                                   \
    && git submodule update --init --recursive cpp                                     \
    && cd /root/Tars/cpp                                                               \
    && mkdir -p build                                                                  \
    && cd build                                                                        \
    && cmake ..                                                                        \
    && make -j4                                                                        \
    && make install
#    && rm -rf /root/Tars

## Install tars go
#RUN go env -w GOPROXY=https://goproxy.io,direct
#RUN go get github.com/TarsCloud/TarsGo/tars \
#    && cd $GOPATH/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go \
#    && go build .  \
#    && mkdir -p /usr/local/go/bin \
#    && chmod a+x /usr/local/go/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go/tars2go \
#    && ln -s /usr/local/go/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go/tars2go /usr/local/go/bin/tars2go

COPY files/template/tarsbuilder/root/bin/entrypoint.sh /bin/
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
