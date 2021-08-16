FROM node:10-stretch-slim AS First
COPY files/sources.list /etc/apt/sources.list
COPY files/binary/tars2case /usr/local/tars/cpp/tools/tars2case
COPY files/template/tarsweb/root /
COPY files/TarsWeb /tars-web

# 设置时区
RUN  ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN  echo Asia/Shanghai > /etc/timezone

RUN apt update && apt install python build-essential busybox -y && busybox --install
RUN cd /tars-web && rm -f package-lock.json && npm install --registry=http://registry.upchinaproduct.com && npm install pm -g
# RUN cd /tars-web/client && rm -f package-lock.json && npm install --registry=http://registry.upchinaproduct.com && npm run build

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
