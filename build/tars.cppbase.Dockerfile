#第一阶段
FROM debian:stretch-slim AS First

# COPY files/sources.list /etc/apt/sources.list
COPY files/entrypoint.sh /bin/entrypoint.sh
RUN  chmod +x /bin/entrypoint.sh

# 设置时区
RUN  ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN  echo Asia/Shanghai > /etc/timezone

RUN  apt update
RUN  apt install busybox -y
RUN  busybox --install
# 安装 ssl 证书
RUN  apt install ca-certificates -y

# 安装并使用 libmariadbclient 作为 libmysqlclient 运行库
RUN  apt install libmariadbclient18 -y
RUN  cd /usr/lib/x86_64-linux-gnu && \
     ln -s libmariadbclient.so.18 libmysqlclient.so && \
     ln -s libmariadbclient.so.18 libmysqlclient.so.18 && \
     ln -s libmariadbclient.so.18 libmysqlclient.so.18.0.0 && \
     ln -s libmariadbclient_r.so.18 libmysqlclient_r.so && \
     ln -s libmariadbclient_r.so.18 libmysqlclient_r.so.18 && \
     ln -s libmariadbclient_r.so.18 libmysqlclient_r.so.18.0.0

# 设置别名，兼容使用习惯
RUN echo alias ll=\'ls -l\' >> /etc/bashrc

# 清理多余文件
RUN  apt purge -y
RUN  apt clean all
RUN  rm -rf /var/lib/apt/lists/*
RUN  rm -rf /var/cache/*.dat-old
RUN  rm -rf /var/log/*.log /var/log/*/*.log

#　第二阶段
FROM scratch
COPY --from=First / /
CMD ["/bin/entrypoint.sh"]
