FROM ubuntu:20.04
COPY files/entrypoint.sh /bin/entrypoint.sh
RUN  chmod +x /bin/entrypoint.sh

# 设置时区
RUN  ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN  echo Asia/Shanghai > /etc/timezone

RUN  apt-get clean && apt update
# Get and install nodejs
RUN apt install -y nodejs npm \ 
    && npm install -g npm pm2
RUN  apt install ca-certificates -y
RUN mkdir -p /usr/local/app/tars/

# 设置别名，兼容使用习惯
RUN echo alias ll=\'ls -l\' >> /etc/bashrc

# 清理多余文件
RUN  apt purge -y
RUN  apt clean all
RUN  rm -rf /var/lib/apt/lists/*
RUN  rm -rf /var/cache/*.dat-old
RUN  rm -rf /var/log/*.log /var/log/*/*.log

CMD ["/bin/entrypoint.sh"]