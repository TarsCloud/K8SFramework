FROM ubuntu:20.04

WORKDIR /root/

ENV DEBIAN_FRONTEND=noninteractive

# Install
RUN apt update 

RUN apt install -y curl gnupg gnupg2 gnupg1 git \
    && apt clean \
    && rm -rf /var/cache/apt

# 安装 kubectl helm 部署工具
RUN curl -s https://mirrors.aliyun.com/kubernetes/apt/doc/apt-key.gpg | apt-key add - 
RUN echo "deb https://mirrors.aliyun.com/kubernetes/apt kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list
RUN apt-get update \
    && apt-get install -y kubectl=1.19.8-00 \
    && apt-mark hold kubectl \
    && apt clean \
    && rm -rf /var/cache/apt

RUN curl -O https://tars-thirdpart-1300910346.cos.ap-guangzhou.myqcloud.com/src/helm-v3.5.2-linux-amd64.tar.gz && tar xzf helm-v3.5.2-linux-amd64.tar.gz && mv linux-amd64/helm /usr/local/bin/ && rm helm-v3.5.2-linux-amd64.tar.gz
RUN helm plugin install https://github.com/chartmuseum/helm-push

COPY tools/exec-deploy.sh /usr/bin/
RUN chmod a+x /usr/bin/exec-deploy.sh
