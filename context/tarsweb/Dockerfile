FROM node:lts-bullseye As First
COPY root /
RUN chmod +x /root/usr/local/tars/cpp/tools/tars2case
RUN chmod +x /root/bin/entrypoint.sh
RUN cd /root/tars-web &&rm -rf .git && rm -rf package-lock.json && npm install

FROM node:lts-bullseye
COPY --from=First /root /

ENV LANG C.UTF-8
ENV LANGUAGE C.UTF-8

RUN rm -rf /bin/ls                                                                        \
# image debian:bullseye had "ls bug", we use busybox ls instead                           \
    && apt update                                                                         \
    && apt install                                                                        \
       ca-certificates openssl telnet curl wget default-mysql-client                      \
       iputils-ping vim tcpdump net-tools binutils procps tree                            \
       libssl-dev zlib1g-dev                                                              \
       tzdata locales busybox -y                                                          \
    && busybox --install                                                                  \
    && apt purge -y                                                                       \
    && apt clean all                                                                      \
    && rm -rf /var/lib/apt/lists/*                                                        \
    && rm -rf /var/cache/*.dat-old                                                        \
    && rm -rf /var/log/*.log /var/log/*/*.log                                             \
    && rm -rf /etc/localtime
# /etc/localtime will block container mount /etc/localtime from host

RUN npm install -g pm2
CMD ["/bin/entrypoint.sh"]
