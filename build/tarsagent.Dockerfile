FROM docker:19.03 As First

FROM debian:bullseye

ENV DEBIAN_FRONTEND=noninteractive
# image debian:bullseye had "ls bug", we use busybox ls instead
RUN rm -rf /bin/ls

RUN apt update                                                                         \
    && apt install                                                                     \
    ca-certificates openssl telnet curl wget default-mysql-client                      \
    iputils-ping vim tcpdump net-tools binutils procps tree                            \
    libssl-dev zlib1g-dev libprotobuf-dev libprotobuf-c-dev                            \
    tzdata localepurge                                                                 \
    busybox -y && busybox --install
RUN locale-gen en_US.utf8
ENV LANG en_US.utf8

RUN apt purge -y                                                                       \
    && apt clean all                                                                   \
    && rm -rf /var/lib/apt/lists/*                                                     \
    && rm -rf /var/cache/*.dat-old                                                     \
    && rm -rf /var/log/*.log /var/log/*/*.log

COPY --from=First /usr/local/bin/docker /usr/local/bin/docker
COPY files/binary/tarsagent /usr/local/app/tars/tarsagent/bin/tarsagent
RUN chmod +x /usr/local/app/tars/tarsagent/bin/tarsagent
CMD ["/usr/local/app/tars/tarsagent/bin/tarsagent"]
