# 服务发布示例

本文介绍tars服务如何快速制作成docker镜像, 并发布到K8S集群上.

## 镜像制作以及helm包

tars服务制作镜像可以通过官方提供的```tarscloud/base-compiler```来制作. 方式如下:

- 拉取编译镜像
```
docker pull tarscloud/base-compiler
```
- 进入镜像
```
docker run -it -v/var/run/docker.sock:/var/run/docker.sock -v:${服务源码目录}:/data/src tarscloud/base-compiler bash
```
- tars服务的yaml文件

每个服务都需要一个(或者多个)values.yaml文件, 来描述服务是如何部署在TarsK8S上面的, 请参考[yaml文件格式](./helm.md)

- 制作tars服务的镜像

在base-compiler容器内部, 使用exec-deploy.sh制作tars服务的镜像, 示例如下:
```
exec-deploy.sh LANG(cpp/nodejs/java-war/java-jar/go/php) Files YamlFile Namespace Tag
```

例如:
```
exec-deploy.sh cpp build/StorageServer yaml/value.yaml tars-dev ruanshudong v1.0.0
```

执行完脚本后会生成:
- 服务的镜像
- helm包路径