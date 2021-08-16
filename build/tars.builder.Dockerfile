FROM golang:1.14-stretch As First
COPY files/sources.list  /etc/apt/sources.list
COPY files/template/tarsbuilder/root /
RUN  apt update && apt install g++ bison flex make cmake zlib1g-dev -y && go env -w GOPROXY=https://goproxy.io,direct
CMD ["/bin/entrypoint.sh"]
