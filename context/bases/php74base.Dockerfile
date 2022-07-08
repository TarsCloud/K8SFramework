FROM php:7.4.26-apache-bullseye AS First
COPY root /
# image debian:bullseye had "ls bug", we use busybox ls instead
RUN rm -rf /bin/ls

RUN apt update                                                                            \
    && apt install git libssl-dev zlib1g-dev busybox -y                                   \
    && busybox --install

RUN yes ''| pecl install igbinary zstd redis swoole                                       \
    && echo "extension=igbinary.so" > /usr/local/etc/php/conf.d/igbinary.ini              \
    && echo "extension=zstd.so" > /usr/local/etc/php/conf.d/zstd.ini                      \
    && echo "extension=redis.so" > /usr/local/etc/php/conf.d/redis.ini                    \
    && echo "extension=swoole.so" > /usr/local/etc/php/conf.d/swoole.ini

RUN cd /root                                                                              \
    && git clone https://github.com/TarsPHP/tars-extension.git                            \
    && cd /root/tars-extension                                                            \
    && phpize                                                                             \
    && ./configure --enable-phptars                                                       \
    && make                                                                               \
    && make install                                                                       \
    && echo "extension=phptars.so" > /usr/local/etc/php/conf.d/phptars.ini


FROM php:7.4.26-apache-bullseye
COPY root /
COPY --from=First /usr/local /usr/local

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

RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
