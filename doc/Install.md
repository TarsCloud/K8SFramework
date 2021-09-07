# TarsK8S 安装

## 在K8S上安装TARS框架
  TarsK8S 以 helm 包的形式对外发布.每个 Helm 包包含了 完成的框架服务. 

  在安装时,以k8s命名空间为分割,每个命名空间可以且只可以部署一套框架服务,每套框架服务都有完整的功能组件,与其他命名空间的框架互相独立.

安装步骤说明:
- 编译代码
```sh
./buildBinary.sh
```

- 制作docker包并推送到仓库

```sh
./buildHelm.sh ${harbor} ${docker_user} ${docker_pass} ${version} 
```

参数说明:
- harbor: 仓库地址
>- 如果是公共仓库, 可以输入: index.docker.io/username, tars公共docker路径为: index.docker.io/tarscloud
>- 如果是私有仓库, 可是使用子路径: xxx.xxx.com/tars-k8s, tars-k8s是项目路径
>- 生成的依赖docker, 都会推送到这个路径下
- docker_user: 访问仓库的用户名
- docker_pass: 访问仓库的密码
- version: 当前编译的版本号

- 安装控制器
```
helm install tarscontroller --set 'helm.dockerhub.registry=${harbor},helm.build.id=${version} ' charts/tarscontroller-${version}.tgz
```

- 在k8s中创建访问仓库的secret
```
kubectl create secret docker-registry tars-image-secret -n tars-test --docker-server=${harbor} --docker-username=${docker_user} --docker-password=${docker_pass}   
```

- 创建TARS环境
```
kubectl create ns tars-test

helm install tars-test -n tars-test --set 'dockerRegistry=${harbor},dockerSecret=tars-image-secret,els.nodes=els.nodes=tars-elasticsearch:9200,helm.build.id=${version},helm.dockerhub.registry=${harbor},web=${web_host}' install/tarsframework-${version}.tgz

```

参数说明:
- tars-test: 这个是K8S上的名字空间 , 表示Tars部署在这个名字空间内
- web_host: 访问tars web的地址, 注意集群中必须已经按了ingress, 且web_host指向了ingress的入口!

## 给节点打标签

通常K8S集群里面节点比较多, 而TARS可能只需要使用里面部分节点, 为了控制到底TARS服务可以运行在哪些节点上, 需要给节点打上特定的标签如下, 分两个层级:

- 如果希望部署的TARS框架以及后续发布的tars服务能被调度到某些节点, 则这些节点需要打上如下标签:

````tars.io/node.$namespace ```, 比如: ```tars.io/node.tars-test```

**这里$namespace 是K8S的名字空间, 即上面helm安装TARS时指定的名字空间**

使用命令行打标签如下:
```
kubectl label nodes $node-name tars.io/node.tars-test=
```

**注意必须打好标签, tars-test这个名字的空间的所有TARS服务才会被调度上去, 这一步必须手工执行!**

- TARS框架本身也有应用的概念, 可以通过给节点继续打标签, 保证TARS某个应用下所有服务能调度到这些节点, 如下方式:

```tars.io/ability.$namespace.$app ```

注意:
>- $namespace 是K8S的名字空间, 即上面helm安装TARS时指定的名字空间
>- $app是TARS下面的应用概念, 即TARS某个应用下的服务可以部署在这些节点(这一步可以在web上控制亲和性)

**完成上述两步以后, 可以在K8S上看到, 除了es和tarslog之外, 其他服务已经启动了**

es和tarslog由于需要存储空间, 因此需要特别处理.

- 给es和tarslog分配LocalPV

目前的K8STARS支持LocalPV, 服务需要LocalPV时, 可以申请, 后续文档会介绍. 

这里Ees和tarslog由于没有合适的PV一直处于pending中, 只需给可以申请LocalPV的节点打上标签```tars.io/SupportLocalVolume ```

使用命令行打标签如下:
```
kubectl label nodes $node-name tars.io/SupportLocalVolume=
```

注意这里的node-name必须已经打上了```tars.io/node.tars-test=```标签, 否则无效!

至此, TARS看框架已经安装完成, 可以打开: ${web_host} 看到tars web管理平台!

## 升级说明

如果升级TARS版本, 按照上述流程重新执行即可, 需要注意的是, 如果TARS框架的crd文件做了变更(install/tarscontroller/crds), 则需要自己手工执行以下, helm upgrade不会再执行crd文件!

示例脚本如下:

```
helm upgrade tarscontroller --set 'helm.dockerhub.registry=${harbor},helm.build.id=${version} ' charts/tarscontroller-${version}.tgz

helm upgrade tars-test -n tars-test --set 'dockerRegistry=${harbor},dockerSecret=tars-image-secret,els.nodes=els.nodes=tars-elasticsearch:9200,helm.build.id=${version},helm.dockerhub.registry=${harbor},web=${web_host}' install/tarsframework-${version}.tgz

```

## 调试

)