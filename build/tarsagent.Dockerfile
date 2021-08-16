FROM docker:19.03 As First

#　第二阶段
FROM tars.cppbase As Second
COPY files/binary/tarsagent /usr/local/app/tars/tarsagent/bin/tarsagent
COPY --from=First /usr/local/bin/docker /usr/local/bin/docker
RUN  chmod +x /usr/local/app/tars/tarsagent/bin/tarsagent

#　第二阶段
FROM scratch
COPY --from=Second / /
CMD ["/usr/local/app/tars/tarsagent/bin/tarsagent"]
