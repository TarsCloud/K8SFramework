FROM ubuntu:20.04
COPY files/template/tarsregistry/root/bin/entrypoint.sh /bin/
COPY files/template/tarsregistry/root/usr/local/app /usr/local/app/

COPY files/binary/tarsregistry /usr/local/app/tars/tarsregistry/bin/tarsregistry
RUN  chmod +x /usr/local/app/tars/tarsregistry/bin/tarsregistry

CMD ["/bin/entrypoint.sh"]
