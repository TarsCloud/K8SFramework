FROM debian:bullseye

# image debian:bullseye had "ls bug", we use busybox ls instead
RUN rm -rf /bin/ls

# /etc/localtime as a solt link will block container mount /etc/localtime from host
RUN rm -rf /etc/localtime

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

COPY files/template/tarscontroller/root /
COPY files/binary/tarscontroller /usr/local/app/tars/tarscontroller/bin/tarscontroller
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
