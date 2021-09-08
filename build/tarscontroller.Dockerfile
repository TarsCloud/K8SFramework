FROM ubuntu:20.04

COPY files/template/tarscontroller/root/bin/entrypoint.sh /bin/
COPY files/template/tarscontroller/root/etc/ /etc/

COPY files/binary/tarscontroller /usr/local/app/tars/tarscontroller/bin/tarscontroller
RUN  chmod +x /usr/local/app/tars/tarscontroller/bin/tarscontroller

CMD ["/bin/entrypoint.sh"]
