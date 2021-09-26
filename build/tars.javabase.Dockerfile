FROM ubuntu:20.04
COPY files/entrypoint.sh /bin/entrypoint.sh
RUN  chmod +x /bin/entrypoint.sh

# 设置时区
RUN  ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN  echo Asia/Shanghai > /etc/timezone

RUN  apt update
RUN  apt install -y openjdk-8-jdk maven telnet curl wget iputils-ping vim tcpdump net-tools
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

CMD ["/bin/entrypoint.sh"]
