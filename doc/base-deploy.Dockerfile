FROM devth/helm:v3.7.1 AS ihelm
RUN mv $(command -v  helm) /tmp/helm

FROM bitnami/kubectl:1.20 AS ikubectl
RUN mv $(command -v  kubectl) /tmp/kubectl

FROM debian:bullseye

# image debian:bullseye had "ls bug", we use busybox ls instead
RUN rm -rf /bin/ls

RUN apt update                                                                         \
    && apt install -y curl gnupg gnupg2 gnupg1 git                                     \
    ca-certificates openssl telnet curl wget                                           \
    iputils-ping vim tcpdump net-tools binutils procps tree python                     \
    busybox && busybox --install

RUN apt purge -y                                                                       \
    && apt clean all                                                                   \
    && rm -rf /var/lib/apt/lists/*                                                     \
    && rm -rf /var/cache/*.dat-old                                                     \
    && rm -rf /var/log/*.log /var/log/*/*.log

COPY --from=ihelm /tmp/helm /usr/bin/helm
COPY --from=ikubectl /tmp/kubectl /usr/bin/kubectl

RUN helm plugin install https://github.com/chartmuseum/helm-push

COPY tools/exec-deploy.sh /usr/bin/
COPY tools/exec-helm.sh /usr/bin/

RUN chmod a+x /usr/bin/exec-deploy.sh
RUN chmod a+x /usr/bin/exec-helm.sh

RUN echo "#!/bin/bash" > /bin/start.sh && echo "while true; do sleep 10; done" >> /bin/start.sh && chmod a+x /bin/start.sh

ENTRYPOINT ["/bin/start.sh"]
