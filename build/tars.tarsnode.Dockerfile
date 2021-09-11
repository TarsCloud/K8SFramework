FROM ubuntu:20.04

COPY files/template/tarsnode/root/bin/entrypoint.sh /bin/
RUN mkdir -p /tarsnode
COPY files/template/tarsnode/root/tarsnode/ /tarsnode/
COPY files/binary/tarsnode /tarsnode/bin/
RUN  chmod +x /tarsnode/bin/tarsnode

CMD ["/bin/entrypoint.sh"]
