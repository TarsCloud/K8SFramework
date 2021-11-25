FROM debian:bullseye
COPY files/template/tarsregistry/root /
COPY files/binary/tarsregistry /usr/local/app/tars/tarsregistry/bin/tarsregistry
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
