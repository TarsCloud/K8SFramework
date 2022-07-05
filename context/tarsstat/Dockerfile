ARG REGISTRY_URL
ARG BUILD_VERSION

FROM debian:bullseye
ARG TARGETPLATFORM
ARG BINARY
COPY /binary /binary
RUN case ${TARGETPLATFORM} in \
         "linux/amd64")  EXECUTE=${BINARY}_linux_amd64  ;; \
         "linux/arm64")  EXECUTE=${BINARY}_linux_arm64  ;; \
    esac \
    && mv /binary/${EXECUTE} /binary/${BINARY}

FROM $REGISTRY_URL/tars.cppbase:$BUILD_VERSION
ARG BINARY
COPY --from=0 /binary/${BINARY} /usr/local/server/bin/${BINARY}
