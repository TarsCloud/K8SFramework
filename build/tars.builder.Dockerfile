# FROM golang:1.14-stretch As First
FROM ubuntu:20.04
# COPY files/sources.list  /etc/apt/sources.list
ENV DEBIAN_FRONTEND=noninteractive
ENV GOPATH=/usr/local/go

# Install
RUN apt update 

RUN apt install -y mysql-client git build-essential unzip make golang cmake flex bison \
    && apt install -y libprotobuf-dev libprotobuf-c-dev zlib1g-dev libssl-dev \
    && apt install -y curl wget net-tools iproute2 \
    && apt clean \
    && rm -rf /var/cache/apt

# Clone Tars repo 
RUN cd /root/ && git clone https://github.com/TarsCloud/Tars.git 

# Install tars go
RUN go get github.com/TarsCloud/TarsGo/tars \
    && cd $GOPATH/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go \
    && go build .  \
    && mkdir -p /usr/local/go/bin \
    && chmod a+x /usr/local/go/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go/tars2go \
    && ln -s /usr/local/go/src/github.com/TarsCloud/TarsGo/tars/tools/tars2go/tars2go /usr/local/go/bin/tars2go 

RUN cd /root/Tars/ \
    && git submodule update --init --recursive cpp \ 
    && cd /root/Tars/cpp \
    && mkdir -p build \
    && cd build \
    && cmake .. \
    && make -j4 \
    && make install

COPY files/template/tarsbuilder/root/bin/entrypoint.sh /bin/

# RUN go env -w GOPROXY=https://goproxy.io,direct

CMD ["/bin/entrypoint.sh"]
