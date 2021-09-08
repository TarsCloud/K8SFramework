FROM ubuntu:20.04
# FROM openjdk:8-slim-stretch AS First
# COPY files/sources.list /etc/apt/sources.list
COPY files/entrypoint.sh /bin/entrypoint.sh
RUN  chmod +x /bin/entrypoint.sh

# 设置时区
RUN  ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN  echo Asia/Shanghai > /etc/timezone

# RUN  apt install gnupg && apt-key adv --keyserver keyserver.ubuntu.com --recv-keys  648ACFD622F3D138
# RUN  apt-key adv --keyserver keyserver.ubuntu.com --recv-keys  0E98404D386FA1D9
RUN  apt update
RUN  apt install -y openjdk-8-jdk maven
RUN  apt install busybox -y
RUN  busybox --install
RUN  apt install ca-certificates -y

# 设置别名，兼容使用习惯
RUN echo alias ll=\'ls -l\' >> /etc/bashrc

# 清理多余文件
RUN  apt purge -y
RUN  apt clean all
RUN  rm -rf /var/lib/apt/lists/*
RUN  rm -rf /var/cache/*.dat-old
RUN  rm -rf /var/log/*.log /var/log/*/*.log

# #　第二阶段
# FROM scratch
# COPY --from=First / /
# ENV JAVA_HOME=/usr/local/openjdk-8
# ENV PATH=${PATH}:${JAVA_HOME}/bin
CMD ["/bin/entrypoint.sh"]
