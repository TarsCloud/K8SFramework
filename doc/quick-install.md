# TarsK8S 快速安装

## 在K8S上安装TARS框架

直接使用官方提供的镜像完成安装! helm请使用helm3版本.

- 添加helm仓库

```
helm repo add tars-k8s https://tarscloud.github.io/K8SFramework/charts
```

- 安装控制器

```
helm install tarscontroller tars-k8s/tarscontroller
```

- 在k8s中创建访问仓库的secret

**tars-image-secret这个名字不要改, helm包等都直接引用这个secret名称, 最好不要修改!**

```
kubectl create secret docker-registry tars-image-secret -n tars-dev --docker-server=${docker_registry} --docker-username=${docker_user} --docker-password=${docker_pass}   
```

- 创建TARS环境
```
kubectl create ns tars-dev

helm install tarsframework -n tars-dev --set 'dockerRegistry=tarscloud,web=${web_host}' tars-k8s/tarsframework

```

参数说明:
- tars-dev: 这个是K8S上的名字空间 , 表示Tars部署在这个名字空间内
- web_host: 访问tars web的地址, 注意集群中必须已经按了ingress, 且web_host指向了ingress的入口!


## 升级说明

如果是升级, 方式类似, 使用 helm upgrade命令即可, 比如:

```
helm upgrade tarscontroller tars-k8s/tarscontroller
helm upgrade tarsframework -n tars-dev --set 'dockerRegistry=tarscloud,web=${web_host}' tars-k8s/tarsframework

```

但是注意: 如果是升级, 比如手动执行crd!!!
```
cd install/tarscontroller/crds
kubectl apply -f ....yaml

```

## 调度

完成以上步骤以后, 实际框架并没有在K8S上调度起来, 请参考[框架调度](./framework-affinity.md)