FROM tars.cppbase AS First
RUN apt update && apt install curl jq -y

# 清理多余文件
RUN  apt purge -y
RUN  apt clean all
RUN  rm -rf /var/lib/apt/lists/*
RUN  rm -rf /var/cache/*.dat-old
RUN  rm -rf /var/log/*.log /var/log/*/*.log

#　第二阶段
FROM scratch
COPY --from=First / /
