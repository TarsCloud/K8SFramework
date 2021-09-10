FROM ubuntu:20.04
COPY files/template/tarsimage/root/bin/entrypoint.sh /bin/
COPY files/binary/tarsagent /usr/local/app/tars/tarsagent/bin/tarsagent

RUN apt update

# 安装docker
RUN apt install -y openssl libssl-dev apt-transport-https ca-certificates curl gnupg2 software-properties-common
RUN curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | apt-key add -
RUN add-apt-repository \
    "deb [arch=amd64] https://mirrors.aliyun.com/docker-ce/linux/ubuntu \
    $(lsb_release -cs) \
    stable"
RUN apt update
RUN apt install -y docker-ce

RUN  chmod +x /usr/local/app/tars/tarsagent/bin/tarsagent

CMD ["/usr/local/app/tars/tarsagent/bin/tarsagent"]
