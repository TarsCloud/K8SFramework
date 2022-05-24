# 构建

当前, 您可以  **TarsCloud K8SFramework** 已经使用 Makefile构建项目. 您可以通过执行 make [target] 打成构建目标,
一些目标可能需要您提供构建参数, 您可以以环境变量的方式指定这些参数,或者在执行 make [target] 时附属参数名=参数值

## 构建参数

在开始介绍构建目标前, 我们希望您能理解一些必要的参数含义

+ BUILD_VERSION
  在构建 **TarsCloud K8SFramework** 过程中, 会生成多种 Docker 镜像 , Makefile 会尝试使用 BUILD_VERSION 作为 镜像的 Tag
  除了通过环境变量或者命令行指定参数外，您可以在 param.sh 填充 \_BUILD_VERSION_ 值来指定参数值

+ REGISTRY_URL , REGISTRY_USER, REGISTRY_PASSWORD
  在构建 **TarsCloud K8SFramework** 过程中, 会生成多种 Docker 镜像 ，Makefile 会尝试将这些镜像推送到您指定的仓库地址 ${REGISTRY_URL},
  如果您的仓库地址开启了账号，密码验证, 您同时需要提供 账号${REGISTRY_USER}, 密码 ${REGISTRY_PASSWORD}
  除了通过环境变量或者命令行指定参数外，您可以在 param.sh 填充 \_REGISTRY_URL_, \_REGISTRY_USER_ \_REGISTRY_PASSWORD_ 值来指定参数值.
  如果您期望编译出的镜像, chart 被实际使用, 此时最好是填充一个公开仓库地址.。当然,使用私密地址也可以工作, 但是在安装时需要额外指定参数.

+ CHART_VERSION, CHART_APPVERSION, CHART_DST
  如果您的构建目标是chart.controller, chart.framework 您需要提供 CHART_VERSION, CHART_APPVERSION,CHART_DST参数值
  分别定义生成 chart 包的 version, appversion 值以及 chart 包的存放目录
  除了通过环境变量或者命令行指定参数外，您可以在 param.sh 填充 \__CHART_VERSION_, \_CHART_APPVERSION_ ,\_CHART_DST_ 值来指定参数值

+ UPLOAD_REGISTRY, UPLOAD_SECRET
  **TarsCloud K8SFramework** 内置了镜像编译服务将您的原生程序包编译成 Docker镜像, 您需要提供一个镜像仓库地址来接收和存储这些镜像
  UPLOAD_REGISTRY 参数值表示您准备的仓库地址
  如果您的仓库需要账号,密码访问, 您需要新建一个 Kubernetes Docker-Registry Secret ,将 Secret 名字赋值给 UPLOAD_SECRET

## 构建目标

+ [base name]
  base name 是 **TarsCloud K8SFramework** 项目中基础镜像的泛指, 具体包括

  ```
  base: cppbase javabase nodejsbase php74base
  ```
  您可以执行 make cppbase , make javabase 等来构建和上传基础镜像
+ [server name]
  server name 是 **TarsCloud K8SFramework** 项目中所有服务的泛指, 具体包括

  ```shell
  controlelr servers: tarscontroller tarsagent
  framework servers: tarskaniko tarsimage tarsnode tarsregistry tarsconfig tarslog tarsnotify tarsstat tarsproperty
                     tarsquerystat tarsqueryproperty tarskevent tarsweb elasticsearch"
  ```
  您可以执行 make tarscontroller , make tarsnode 等在构建和上传这些服务镜像
+ chart.controller, chart.framework
  分别构建 Controller, Framework 的 Helm Chart
+ secret
  用于构建 一个 Kubernetes docker-registry secret,
  除了仓库地址,仓库账号,仓库密码外，您还需要提供命名空间参数( NAMESPACE) 以及 Secret 的名字参数(NAME)
+ install.controller ,upgrade.controller
  分别用于安装和升级 Controller
  您需要提供 CHART 参数，指向 Controller Chart 包
  如果您构建 Chart 包时使用的是私用仓库地址,你需要使用 "make secret " 在 tars-system 命名空间新建一个 docker-registry secret , 并 额外指定参数 CONTROLLER_SECRET=${secret name}

+ install.framework ,upgrade.framework
  分别用于安装和升级 framework
  您需要提供 CHART 参数，表示 Framework Chart 的路径
  您需要提供 命名空间参数, 表示 Framework Chart 安装的命名空间
  您需要提供 UPLOAD_REGISTRY, UPLOAD_SECRET 参数
  如果您构建 Chart 包时使用的是私用仓库地址,你需要使用 "make secret " 在安装命名空间新建一个 docker-registry secret , 并额外指定参数 FRAMEWORK_SECRET=${secret name}

+ base , controller , framework
  分别为 基础镜像, Controller Server , Framework Server 的集合构建目标
+ chart
  chart.controller , chart.framework 的集合构建目标
+ all
  base , controller , framework, all 的集合构建目标