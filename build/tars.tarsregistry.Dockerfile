FROM tars.cppbase As First
COPY files/template/tarsregistry/root /
COPY files/binary/tarsregistry /usr/local/app/tars/tarsregistry/bin/tarsregistry
RUN  chmod +x /usr/local/app/tars/tarsregistry/bin/tarsregistry
#　第二阶段
FROM scratch
COPY --from=First / /
CMD ["/bin/entrypoint.sh"]
