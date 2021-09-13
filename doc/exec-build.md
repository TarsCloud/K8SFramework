

# 镜像以及helm包制作

为了方便开发者制作tars服务的镜像, 以及对应版本的helm包, 特提供了一个```exec-build.sh```来完成, 该脚本已经被内置到编译镜像中了, 你可以在编译镜像中直接使用!

该脚本的使用如下:
```
exec-build.sh BaseImage SERVERTYPE(cpp/nodejs/java-war/java-jar/go/php) Files YamlFile Namespace Registry Tag Push(true/false) Dockerfile
```

参数说明:
- BaseImage: 依赖的基础镜像(官方镜像为: tarscloud/tars.cppbase:$tag)
- SERVERTYPE: 语言, 目前支持: cpp/nodejs/java-war/java-jar/go/php
- Files: 需要打包进docker中的文件或者目录
- YamlFile: yaml文件描述服务用, 可以参考[helm包](./helm.md)
- Namespace: k8s上的名字空间, 安装Tars时指定的
- Registry: 镜像仓库的地址, 最后生成的镜像是: $Registry/$APP/$SERVER:$TAG
- Tag: 版本号, 格式必须符合版本号规范: vx.x.x, 例如v1.0.2
- Push: 制作好的docker是否push到仓库中($Registry/$APP/$SERVER:$TAG)
- Dockerfile: 制作镜像的dockerfile路径, 正常情况不需要提供, 你如果希望自己改写Dockerfile, 则需要提供, 请参考[Dockerfile](../Dockerfile.md)
例如:
```
exec-build.sh tarscloud/tars.cppbase:v1.0.0 cpp build/StorageServer yaml/value.yaml tars-dev xxx.harbor.com v1.0.0
```

执行完脚本后会生成:
- 服务的镜像: $Registry/$APP/$SERVER:$TAG, 可以通过```docker images```查看到, 你需要自己推送到docker仓库
- helm包: $APP-$SERVER-$TAG.tgz, 该helm包对应了当前镜像, 该helm压缩文件会被生成在当前文件夹, 你可以自己把helm包推送到自己的charts仓库

为了方便你部署, 提供了```exec-deploy```脚本, 方便你部署到K8S集群, 具体请参考[exec-deploy](./exec-deploy.md)
