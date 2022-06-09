# 开发

本文是介绍 **TarsCloud K8SFramework** 项目自身的开发流程,而不是介绍 Tars 业务服务的开发流程.  
Tars 业务服务开发您可以参考 [TarsDoc](https://tarscloud.github.io/TarsDocs/base/tars-intro.html)

## 文件及目录构成
**TarsCloud K8SFramework** 项目的主要目录构成:
+ src: 所有服务程序及其依赖的第三方源代码
+ context: 待构建的 Docker镜像上下文
+ helm: 待构建的 Helm Chart 的模板文件
+ charts: 已发布的 Helm Chart 文件
+ t: 自动化测试需要的代码及文件

## 开发
