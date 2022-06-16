FROM elasticsearch:7.11.2
COPY root /
RUN chown elasticsearch:root /usr/share/elasticsearch/config -R
RUN rm -rf /etc/localtime
