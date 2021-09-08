# FROM docker:19.03 As First

# 　第二阶段
FROM tars.cppbase

# FROM ubuntu:20.04
COPY files/template/tarsimage/root /
COPY files/binary/tarsimage /usr/local/app/tars/tarsimage/bin/tarsimage

RUN apt update

# 安装docker
RUN apt install -y apt-transport-https ca-certificates curl gnupg2 software-properties-common
RUN curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | apt-key add -
RUN add-apt-repository \
    "deb [arch=amd64] https://mirrors.aliyun.com/docker-ce/linux/ubuntu \
    $(lsb_release -cs) \
    stable"
RUN apt update
RUN apt install -y docker-ce

# COPY --from=First /usr/local/bin/docker /usr/local/bin/docker
RUN  chmod +x /usr/local/app/tars/tarsimage/bin/tarsimage

# #　第三阶段
# FROM scratch
# COPY --from=Second / /
CMD ["/bin/entrypoint.sh"]
