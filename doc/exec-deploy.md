

# 执行部署

## 脚本说明 
为了方便你部署, 提供了```exec-deploy.sh```脚本, 方便你部署到K8S集群, 具体请参考[exec-deploy](./exec-deploy.md)

该脚本也被内置到tarscloud/base-compiler镜像中!

该脚本的使用如下:
```
exec-deploy.sh Namespace HelmPackage
```

参数说明:
- Namespace: k8s上的名字空间, 安装Tars时指定的
- HelmPackage: exec-build.sh脚本生成的helm包 
例如:
```
exec-deploy.sh tars-dev od-storageserver-v1.0.0.tgz
```

## 特殊说明

exec-build.sh 承担了镜像/helm生成的功能, exec-deploy.sh 承担了服务发布的功能, 但是镜像/helm包推送到对应仓库, 以及发布到K8S集群, 需要K8S的config, 这两点是需要业务自己处理的.

因此建议做法, 是自己基于tarscloud/base-compiler制作一个Docker, 内置仓库, 以及K8S集群的config等信息, 在自动CI/CD脚本中, 通过调用exec-build.sh & exec-deploy.sh 来完成自动化!