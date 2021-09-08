#  Dockerfile说明

## 目的

部署在K8S上每个业务最终都要编译成Docker, 才能发布到K8S+TARS上.

每个业务制作成docker, 都有确定的规范, 必须遵循这个规范构建docker, 因为:
- 实际tars服务制作成docker后, 跑在k8s上docker内部都有两个进程, 一个业务进程, 一个tarsnode进程;
- tarsnode进程是pod启动时外部注入进去的, 制作docker时候并不具备
- 业务进程目录结构有一定的规范, 必须在: ```/usr/local/server/bin``` 下
- 业务进程其实是tarsnode进程拉起的
- 业务进程都有类型: cpp/nodejs/java-war/java-jar/go/php
- tarsnode根据业务类型才能启动服务

**注意Dockerfile通常不需要你自己实现, 统一制作了一个docker容器, 里面存在了各种语言制作的Dockerfile, 从容器中copy出来即可**

## 格式示例

Dockerfile格式大致介绍下, 以cpp语言的为例

cpp格式如下, 其他格式, 更换ServerType即可, 具体可以参考Dockerfile目录:

```dockerfile
FROM tarscloud/tars.cppbase
ENV ServerType=cpp

ARG BIN
RUN mkdir -p /usr/local/server/bin/
COPY $BIN /usr/local/server/bin/

```

**注意, 如果你自己的私有仓库tarscloud/tars.cppbase, 这个地址你需要自己更换!**


- 制作docker

正常情况docker的制作, 是在自动构建脚本中构建, 并推送到harbor仓库中的同时发布, 当然你也可以手工制作docker, 比如:
```
docker build . -t xxx/notifyproxyserver --build-arg BIN=build/bin/NotifyProxyServer
docker push xxx/notifyproxyserver
```

## 说明
>- 当一个git工程只编译出一个服务时, Dockerfile中BIN这个参数你写死就可以了
>- 当一个git工程编译出多个二进制服务时, 可以用上面那个传参机制(分别传入不同的可执行程序)构建出多个docker



