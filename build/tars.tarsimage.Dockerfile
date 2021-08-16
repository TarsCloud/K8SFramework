FROM docker:19.03 As First

#　第二阶段
FROM tars.cppbase As Second
COPY files/template/tarsimage/root /
COPY files/binary/tarsimage /usr/local/app/tars/tarsimage/bin/tarsimage
COPY --from=First /usr/local/bin/docker /usr/local/bin/docker
RUN  chmod +x /usr/local/app/tars/tarsimage/bin/tarsimage

#　第三阶段
FROM scratch
COPY --from=Second / /
CMD ["/bin/entrypoint.sh"]
