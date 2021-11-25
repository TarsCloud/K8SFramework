FROM php:7.4.26-apache-bullseye As First

RUN yes ''| pecl install igbinary zstd redis swoole                                    \
    && echo "extension=igbinary.so" > /usr/local/etc/php/conf.d/igbinary.ini           \
    && echo "extension=zstd.so" > /usr/local/etc/php/conf.d/zstd.ini                   \
    && echo "extension=redis.so" > /usr/local/etc/php/conf.d/redis.ini                 \
    && echo "extension=swoole.so" > /usr/local/etc/php/conf.d/swoole.ini

RUN apt update                                                                         \
    && apt install git libssl-dev zlib1g-dev -y                                        \
    && cd /root                                                                        \
    && git clone https://github.com/TarsPHP/tars-extension.git                         \
    && cd /root/tars-extension                                                         \
    && phpize                                                                          \
    && ./configure --enable-phptars                                                    \
    && make                                                                            \
    && make install                                                                    \
    && echo "extension=phptars.so" > /usr/local/etc/php/conf.d/phptars.ini


FROM php:7.4.26-apache-bullseye

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

COPY --from=First /usr/local /usr/local
COPY files/entrypoint.sh /bin/entrypoint.sh
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
