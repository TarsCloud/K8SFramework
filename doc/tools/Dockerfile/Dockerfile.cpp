FROM tarscloud/tars.cppbase

#注意这里是cpp, 如果是其他语言请更换成对应go/java-war/java-jar/nodejs等
ENV ServerType=cpp

ARG BIN
RUN mkdir -p /usr/local/server/bin/
COPY $BIN /usr/local/server/bin/

