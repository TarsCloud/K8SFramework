ARG BaseImage
ARG ServerType
ARG BIN

FROM ${BaseImage}

ENV ServerType=${ServerType}

RUN mkdir -p /usr/local/server/bin/
COPY $BIN /usr/local/server/bin/

