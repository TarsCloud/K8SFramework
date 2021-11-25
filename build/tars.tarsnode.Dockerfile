FROM debian:bullseye
COPY files/template/tarsnode/root /
COPY files/binary/tarsnode /tarsnode/bin/tarsnode
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]