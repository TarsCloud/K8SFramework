# 如何调试

## 基本思路
当服务部署在K8S集群中以后, 如何快速简单的调试是大的问题!

最理想的做法, 我们当前开发和运行的服务, 就在K8S内部, 这样可以畅通无阻的访问任何K8S服务, 那调试起来肯定是最简单的!

如何能做到这个效果呢? 思路如下:
- 在K8S上启动一个Pod
- 这个Pod具备编译和运行tars服务的环境
- 这个Pod挂载宿主机文件系统, 代码放在这个目录下, 这样即使Pod挂了漂移了也不影响
- Pod做到不漂移
- Pod使用K8SFramework的LocalPv来实现

具体的yaml文件如下:
```yaml
apiVersion: k8s.tars.io/v1beta1
kind: TServer
metadata:
  labels:
    tars.io/ServerApp: tars
    tars.io/ServerName: compiler
    tars.io/SubType: normal
  name: tars-compiler
  namespace: tars-dev
spec:
  app: tars
  server: compiler
  important: 3
  subType: normal
  normal:
    ports:
      - isTcp: true
        name: http
        port: 8080
  k8s:
    mounts:
      - mountPath: /data
        name: data
        readOnly: false
        source:
          tLocalVolume: {}
    replicas: 1
  release:
    source: tars-compiler
    id: v-0000001
    image:
      tarscloud/base-compiler
```

说明:
- 注意名字空间, 这里为tars-dev,  必须你和安装在K8S的tars框架相同的名字空间
- replicas可以指定副本数, 可以给每个人分配一个, 相当于开发机了
- tarscloud/base-compiler 是编译环境, 可以制作编译环境
- id: v-0000001, 每次更新编译环境, 需要更新这个id值, 不冲突即可
- tars.io/SubType: normal 以及 subType: normal, 这里值是normal, 表示非tars服务的pod, 目前在tarsweb上看不到这类服务

## 使用步骤

部署tars-compiler

```sh
kubectl apply -f debug.yaml
```

容器在K8S启动以后, 容器内部的```/data```目录已经映射到宿主机的 ```/usr/local/app/tars/host-mount/tars-dev/tars.compiler/data```


你可以进入容器开发服务了
```
kubectl exec -it tars-compiler-0 -n tars-dev -- bash
```

这时候服务可以连接集群中任何服务!主控地址为: tcp -h tars-tarsregistry -p 17890

**注意进入容器/data目录开发!**