# TarsK8S说明文档

Tars深度整合K8S中, 主要说明如下:
- 使用了CRD+Operator的方式完成整合, 即可以用kubectl完成对Tars中所有资源的控制, 包括服务, 配置, 发布镜像等
- 安装时首先安装tars控制器服务(tarscontroller), 然后可以在一个k8s集群中安装多个tars框架(每个都有自己独立的namespace)
- tars框架包含了通用的组件, tarsregistry, tarsstat等组件, 以及tarsweb, 即也可以通过tarsweb来管理tars业务服务(tarsweb的版本>=v2.4.18)
- tars框架支持了大量关键特性, 帮助你管理服务的部署, 比如业务服务亲和性设置, localpv等
- 使用文档也包含了CICD的说明, 你可以使用源码中提供好的helm包, 快速完成服务的部署(当然也可以通过tarsweb来完成)

详细文档[请参见](https://tarscloud.github.io/TarsDocs/) k8s部分
