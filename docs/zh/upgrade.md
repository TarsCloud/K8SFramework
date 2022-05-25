# 目录

- [升级](#升级)
    * [版本与兼容性计划](#版本与兼容性计划)
        + [版本格式](#版本格式)
        + [版本规则](#版本规则)
        + [兼容性规则](#兼容性规则)
        + [兼容性流程](#兼容性流程)
    * [下载](#下载)
    * [执行](#执行)
        + [升级 CRD](#升级CRD)
        + [升级 Controller](#升级Controller)
        + [等待 Controller 启动](#等待Controller启动)
        + [升级 Framework](#升级Framework)
        + [等待 Framework 启动](#等待Framework启动)

# 升级

## 版本与兼容性计划

从版本 1.3.2 开始,我们制定了全新的版本与兼容性规则

### 版本格式

**TarsCloud K8SFramework** 在发布 Helm Chart 时,采用与 Kubernetes 相同格式的版本.  
具体为: "主版本号.次版本号.修订号-附注说明", 其中附注说明为可选项.

### 版本规则

1. 根据最高 CRD Version 生成 项目主版本号与次版本号, 比如:
   最高 CRD Version 为 v1beta3 时, 主版本号为 "1" , 次版本号为 "3" ,
   最高 CRD Version 为 v2beta1 时,主版本号为 "2" , 次版本号为 "1"
2. 修订号与附注在发布时酌情定义.
3. 主版本号,次版本号,修订号按整数升序发布

### 兼容性规则

**TarsCloud K8SFramework** 提供最多三个"主版本.次版本"的兼容性保证.具体规则为

1. 相同的 "主版本.次版本", 提供兼容性保证
2. 当 "次版本" == 0 时, 提供前一主版本的最新两个次版本的兼容性保证
3. 当 "次版本" == 1 时, 提供前一主版本的最新一个次版本,同主版本的最新一个次版本的兼容性保证
4. 当 "次版本" >=2 时, 提供同主版本的最新两个次版本的兼容性保证

### 兼容性流程

1. 升级 Controller 时, 请确保待升级的 Controller 版本能兼容已安装的 Framework 版本
2. 升级 Framework  时, 请确保已安装的 Controller 版本能兼容待升级的 Framework 版本
3. 降级 Framework  时, 请确保已安装的 Controller 版本能兼容待降级的 Framework 版本
4. 降级 Controller 可能导致不可预知的问题, 我们不建议您对 Controller 执行降级操作

不满足兼容性流程的 升/降级操作 可能导致 framework 程序运行失败,甚至 crd 对象丢失

## 下载

您可以 <<直接下载>> 或者使用 << helm repo >> 两种方式来获取已发布版本.  
需要注意的是只要符合兼容性规则,您可以分别升级 Controller 或 Framework.

+ 直接下载

您可以在 [github](https://github.com/TarsCloud/K8SFramework/tree/master/charts) 查看并下载 **TarsCloud K8SFramework** 已经发布的 Helm
Chart

+ Helm repo

```shell
helm repo add tars-k8s https://tarscloud.github.io/K8SFramework/charts
helm search repo tars-k8s
```

## 执行

### 升级CRD

如果您意图跨越 "主版本.次版本" 号升级 Controller, 则需要先升级 crd 定义  
同时, 跨越 "主版本.次版本" 号升级 Controller 尤其需要关注版本兼容性,  
不兼容的升级可能导致正在运行的服务中断甚至 crd 对象丢失

```shell
helm show crds tarscontroller-${Version}.tgz > tars-crds.yaml                  # 直接下载
helm show crds tars-k8s/tarscontroller-${Version} > tars-crds.yaml             # helm repo

sed -i -E 's#^apiVersion:(.*)$#---\napiVersion:\1#g' crds.yaml
kubectl apply -f tars-crds.yaml
```

### 升级Controller

您可以执行以下命令来升级 Controller:

```shell 
helm upgrade tarscontroller tarscontroller-${Version}.tgz                      # 直接下载
helm upgrade tarscontroller tars-k8s/tarscontroller-${Version}                 # helm repo
```

### 等待Controller 启动

您可以执行以下命令查看 Controller 启动状态

```shell 
kubectl get pods -n tars-system -o wide                                       
```

### 升级Framework

您可以执行以下命令来升级 Framework

```shell 
helm upgrade tarsframework -n ${Namespace} tarsframework-${Version}.tgz        # 直接下载
helm upgrade tarsframework -n ${Namespace} tars-k8s/tarsframework-${Version}   # helm repo
```

如果您需要变更参数,请新建 tarsframework.yaml 文件,并按说明填充字段

```yaml
# TarsCloud K8SFramework 内置了镜像编译服务,可以将您的原生程序包编译成 Docker镜像,请将您准备镜像仓库地址填充到 upload.registry
# 如果您的镜像仓库地址需要账号密码认证,那就需要新建一个 Kubernetes docker-registry secret,并将 secret 名字填充到 upload.secret
# 新建 docker-registry secret 的指令为: kubectl create secret docker-registry ${secret-name} -n ${namespace} --docker-server=${registry} --docker-username=${user} --docker-password=${pass}
upload:
  registry: ""
  secret: ""

# 如果您的 Kubernetes 集群安装了 Ingress, 可以通过此域名访问 TarsCloud K8SFramework 管理平台
web: ""
```

然后执行以下命令来升级 Framework

```shell
helm upgrade tarsframework -n ${namespace} -f tarsframework.yaml tarsframework-${version}.tgz      #直接下载
helm upgrade tarsframework -n ${namespace} -f tarsframework.yaml tars-k8s/tarsframework-${version} #helm repo
```

### 等待Framework 启动

您可以执行以下命令来查看 Framework 启动状态

```shell
kubeclt get pods -n ${namespace} -o wide
```
