FROM docker:19.03 As First

FROM debian:bullseye

COPY --from=First /usr/local/bin/docker /usr/local/bin/docker
COPY files/template/tarsimage/root /
COPY files/binary/tarsimage /usr/local/app/tars/tarsimage/bin/tarsimage
RUN chmod +x /bin/entrypoint.sh
CMD ["/bin/entrypoint.sh"]
