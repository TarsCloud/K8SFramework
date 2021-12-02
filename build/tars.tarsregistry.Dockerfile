FROM debian:bullseye

ENV DEBIAN_FRONTEND=noninteractive

# image debian:bullseye had "ls bug", we use busybox ls instead
RUN rm -rf /bin/ls

# /etc/localtime as a solt link will block container mount /etc/localtime from host
RUN rm -rf /etc/localtime

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

COPY files/template/tarsregistry/root /
COPY files/binary/tarsregistry /usr/local/app/tars/tarsregistry/bin/tarsregistry
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
