FROM ubuntu:20.04

RUN apt update

RUN apt install -y openssl libssl-dev

COPY files/template/tarscontroller/root/bin/entrypoint.sh /bin/
COPY files/template/tarscontroller/root/etc/ /etc/

COPY files/binary/tarscontroller /usr/local/app/tars/tarscontroller/bin/tarscontroller
RUN  chmod +x /usr/local/app/tars/tarscontroller/bin/tarscontroller

CMD ["/bin/entrypoint.sh"]
