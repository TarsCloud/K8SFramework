FROM node:lts-bullseye As First

COPY files/template/tarsweb/root /root/
COPY files/binary/tars2case /root/usr/local/tars/cpp/tools/tars2case
RUN chmod +x /root/usr/local/tars/cpp/tools/tars2case

RUN apt update                                                                         \
    && apt install git -y                                                              \
    && cd /root                                                                        \
    && git clone https://github.com/TarsCloud/TarsWeb                                  \
    && cd /root/TarsWeb                                                                \
    && rm -f package-lock.json && npm install && npm install pm2 -g                    \
    && mv /root/TarsWeb /root/tars-web

FROM node:lts-bullseye

# image debian:bullseye had "ls bug", we use busybox ls instead
RUN rm -rf /bin/ls

RUN apt update                                                                         \
    && apt install                                                                     \
    ca-certificates openssl telnet curl wget default-mysql-client                      \
    iputils-ping vim tcpdump net-tools binutils procps tree                            \
    libssl-dev zlib1g-dev libprotobuf-dev libprotobuf-c-dev                            \
    busybox -y && busybox --install

RUN apt purge -y                                                                       \
    && apt clean all                                                                   \
    && rm -rf /var/lib/apt/lists/*                                                     \
    && rm -rf /var/cache/*.dat-old                                                     \
    && rm -rf /var/log/*.log /var/log/*/*.log

COPY --from=First /root /
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
