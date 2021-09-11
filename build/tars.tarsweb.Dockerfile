FROM ubuntu:20.04
COPY files/binary/tars2case /usr/local/tars/cpp/tools/tars2case
COPY files/template/tarsweb/root/bin/entrypoint.sh /bin/
COPY TarsWeb /tars-web

# 设置时区
RUN  ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN  echo Asia/Shanghai > /etc/timezone

RUN apt update && apt install nodejs npm python build-essential -y
RUN cd /tars-web && rm -f package-lock.json && npm install && npm install pm2 -g

# 清理多余文件
RUN  apt purge -y
RUN  apt clean all
RUN  rm -rf /var/lib/apt/lists/*
RUN  rm -rf /var/cache/*.dat-old
RUN  rm -rf /var/log/*.log /var/log/*/*.log

CMD ["/bin/entrypoint.sh"]
