# FROM tars.cppbase As First
FROM ubuntu:20.04
COPY files/template/tarscontroller/root /
COPY files/binary/tarscontroller /usr/local/app/tars/tarscontroller/bin/tarscontroller
RUN  chmod +x /usr/local/app/tars/tarscontroller/bin/tarscontroller

#　第二阶段
# FROM scratch
# COPY --from=First / /
CMD ["/bin/entrypoint.sh"]
