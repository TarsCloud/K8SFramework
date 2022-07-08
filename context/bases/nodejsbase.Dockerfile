FROM node:lts-bullseye

ENV LANG C.UTF-8
ENV LANGUAGE C.UTF-8

RUN rm -rf /bin/ls                                                                        \
# image debian:bullseye had "ls bug", we use busybox ls instead                           \
    && apt update                                                                         \
    && apt install                                                                        \
       ca-certificates openssl telnet curl wget default-mysql-client                      \
       gnupg iputils-ping vim tcpdump net-tools binutils procps tree                      \
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

RUN mkdir -p /usr/local/app/tars                                                          \
    && npm install -g @tars/node-agent                                                    \
    && mv /usr/local/lib/node_modules/@tars/node-agent /usr/local/app/tars/               \
    && cd /usr/local/app/tars/node-agent                                                  \
    && npm install

ENV NODE_AGENT_BIN=/usr/local/app/tars/node-agent/bin/node-agent

COPY root /
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
