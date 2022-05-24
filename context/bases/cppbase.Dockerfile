FROM debian:bullseye
COPY root /

ENV LANG en_US.utf8
ENV DEBIAN_FRONTEND=noninteractive

RUN rm -rf /bin/ls                                                                        \
# image debian:bullseye had "ls bug", we use busybox ls instead                           \
    && apt update                                                                         \
    && apt install                                                                        \
       ca-certificates openssl telnet curl wget default-mysql-client                      \
       iputils-ping vim tcpdump net-tools binutils procps tree                            \
       libssl-dev zlib1g-dev                                                              \
       tzdata localepurge busybox -y                                                      \
    && busybox --install                                                                  \
    && locale-gen en_US.utf8                                                              \
    && apt purge -y                                                                       \
    && apt clean all                                                                      \
    && rm -rf /var/lib/apt/lists/*                                                        \
    && rm -rf /var/cache/*.dat-old                                                        \
    && rm -rf /var/log/*.log /var/log/*/*.log                                             \
    && rm -rf /etc/localtime
# /etc/localtime will block container mount /etc/localtime from host

RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
