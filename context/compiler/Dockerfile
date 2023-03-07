FROM golang:1.17-bullseye
COPY root /

RUN apt update                                                                          \
    && apt install make cmake flex bison libssl-dev zlib1g-dev -y                       \
    && apt purge -y                                                                     \
    && apt clean all                                                                    \
    && rm -rf /var/lib/apt/lists/*                                                      \
    && rm -rf /var/cache/*.dat-old                                                      \
    && rm -rf /var/log/*.log /var/log/*/*.log

RUN cd /root                                                                                           \
    && wget https://boostorg.jfrog.io/artifactory/main/release/1.81.0/source/boost_1_81_0.tar.gz       \
    && tar zxvf  boost_1_81_0.tar.gz                                                                   \
    && cd boost_1_81_0 && ./bootstrap.sh && ./b2 link=static runtime-link=static --with-json install   \
    && rm -rf /root/boost_1_81_0.tar.gz /root/boost_1_81_0

RUN cd /root/TarsCpp                                                                    \
    && mkdir -p build                                                                   \
    && cd build                                                                         \
    && cmake ..                                                                         \
    && make -j1                                                                         \
    && make install                                                                     \
    && rm -rf /root/Tarscpp

RUN go env -w GOPROXY=https://proxy.golang.com.cn,direct                                \
    && go env -w GOSUMDB=sum.golang.org

ENTRYPOINT [ "/usr/bin/make","-f","/root/Makefile" ]
