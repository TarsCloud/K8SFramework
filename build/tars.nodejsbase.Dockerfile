FROM node:lts-bullseye

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

RUN mkdir -p /usr/local/app/tars                                                       \
    && npm install -g @tars/node-agent                                                 \
    && mv /usr/local/lib/node_modules/@tars/node-agent /usr/local/app/tars/            \
    && cd /usr/local/app/tars/node-agent                                               \
    && npm install

ENV NODE_AGENT_BIN=/usr/local/app/tars/node-agent/bin/node-agent
COPY files/entrypoint.sh /bin/entrypoint.sh
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
