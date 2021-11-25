FROM debian:bullseye
COPY files/template/tarscontroller/root /
COPY files/binary/tarscontroller /usr/local/app/tars/tarscontroller/bin/tarscontroller
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
