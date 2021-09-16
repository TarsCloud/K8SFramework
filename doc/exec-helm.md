

# 制作Helm包

## 脚本说明 

脚本```exec-helm.sh```提供了制作helm包的功能, 这个脚本的内容其实包含在```exec-build.sh```中了, 但是之所以独立出来, 是因为有时候镜像已经存在(比如官方提供好的网关等镜像), 使用者可以直接使用, 而不再需要自己重新编译源码, 这个时候可以直接使用```exec-helm.sh```针对某镜像制作helm包即可.

该脚本也被内置到```tarscloud/base-compiler``` 以及 ```tarscloud/base-deploy```镜像中, 方便你使用!

脚本的使用如下:
```
exec-helm.sh YamlFile Image
```

参数说明:
- YamlFile: yaml文件地址, [yaml文件请参考](./helm.md), 你可以根据自己安装的情况, 调整values.yaml值
- Image: 镜像地址
例如:
```
exec-helm.sh yaml/value.yaml tarscloud/base.gatewayserver
```
