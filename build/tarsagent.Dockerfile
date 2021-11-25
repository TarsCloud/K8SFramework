FROM docker:19.03 As First

FROM debian:bullseye
COPY --from=First /usr/local/bin/docker /usr/local/bin/docker
COPY files/binary/tarsagent /usr/local/app/tars/tarsagent/bin/tarsagent
RUN chmod +x /usr/local/app/tars/tarsagent/bin/tarsagent
CMD ["/usr/local/app/tars/tarsagent/bin/tarsagent"]
