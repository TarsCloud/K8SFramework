

# 镜像以及helm包制作

为了方便开发者制作tars服务的镜像, 以及对应版本的helm包, 特提供了一个```exec-deploy.sh```来完成, 该脚本以及被内置到编译镜像中了, 你可以在编译镜像中直接使用!

该脚本的使用如下:
```
exec-deploy.sh LANG(cpp/nodejs/java-war/java-jar/go/php) Files YamlFile Namespace Registry Tag Dockerfile
```

参数说明:
- LANG: 语言, 目前支持: cpp/nodejs/java-war/java-jar/go/php
- Files: 需要打包进docker中的文件或者目录
- YamlFile: yaml文件描述服务用, 可以参考[helm包](./helm.md)
- Namespace: k8s上的名字空间, 安装Tars时指定的
- Registry: 镜像仓库的地址, 最后生成的镜像是: $Registry/$APP/$SERVER:$TAG
- Tag: 版本号
- Dockerfile: 制作镜像的dockerfile路径, 正常情况不需要提供, 你如果希望自己改写Dockerfile, 则需要提供, 请参考[Dockerfile](../DockerfileAbout.md)
例如:
```
exec-deploy.sh cpp build/StorageServer yaml/value.yaml tars-dev xxx.harbor.com v1.0.0
```

执行完脚本后会生成:
- 服务的镜像: $Registry/$APP/$SERVER:$TAG
- helm包: 
