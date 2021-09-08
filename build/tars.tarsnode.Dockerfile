# FROM tars.cppbase As First
FROM ubuntu:20.04
COPY files/template/tarsnode/root /
COPY files/binary/tarsnode /tarsnode/bin/tarsnode
RUN  chmod +x /tarsnode/bin/tarsnode

# 　第二阶段
# FROM scratch
# COPY --from=First / /
CMD ["/bin/entrypoint.sh"]
