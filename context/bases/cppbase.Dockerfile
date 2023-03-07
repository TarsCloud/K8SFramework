FROM debian:bullseye

ENV LANG C.UTF-8
ENV LANGUAGE C.UTF-8

RUN apt update                                                                            \
    && apt install                                                                        \
    ca-certificates openssl gdb telnet curl wget default-mysql-client                     \
    gnupg iputils-ping vim tcpdump net-tools binutils procps tree                         \
    libssl1.1 zlib1g-dev                                                                  \
    tzdata locales busybox -y                                                             \
    && busybox --install                                                                  \
    && apt purge -y                                                                       \
    && apt clean all                                                                      \
    && rm -rf /var/lib/apt/lists/*                                                        \
    && rm -rf /var/cache/*.dat-old                                                        \
    && rm -rf /var/log/*.log /var/log/*/*.log                                             \
    && rm -rf /etc/localtime
# /etc/localtime will block container mount /etc/localtime from host

COPY root /
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
