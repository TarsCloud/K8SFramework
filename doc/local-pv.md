
# LocalPV
   
我们的大部分使用场景是自建集群,没有使用网络盘的条件, 因此可以简单配置 LocalPV 非常重要

## tserver中的描述

在自定义资源server中, 有描述字段专门描述LocalPV, 即字段: ```tsever.k8s.mounts.source.TLocalVolume```

如下所示:

```yaml
tLocalVolume:
    type: object
    properties:
    uid:
        type: string
        pattern: ^(0|[1-9][0-9]{0,5})$
        default: "0"
    gid:
        type: string
        pattern: ^(0|[1-9][0-9]{0,5})$
        default: "0"
    mode:
        type: string
        parrern: ^0?[1-7]{3,3}$
        default: "755"
```

示例如下:
```yaml
k8s:
    env:
    - name: Namespace
    valueFrom:
        fieldRef:
        fieldPath: metadata.namespace
    - name: PodName
    valueFrom:
        fieldRef:
        fieldPath: metadata.name
    mounts:
    - name: remote-log-dir
      mountPath: /usr/local/app/tars/remote_app_log
      source:
        tLocalVolume:
          gid: "0"
          mode: "755"
          uid: "1000"
```

以上代码申请了pv操作, 说明:
- 注意实际的pv申请是有tars.tarsagent来操作的(它是TarsK8S框架部署, daemonset类型, 每个节点部署一个, 它会给需要申请的pv的服务创建pv)
- tars服务部署以后, tserver的yaml会被展开如下两段yaml, 以下以tarslog为例:
- /usr/local/app/tars/remote_app_log 是pod内路径, 会映射到宿主机路径中, 请参考后续说明

### LocalPV对应的pvc

pvc是由tarscontroller根据tserver的yaml来展开得到的.

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
    annotations:
        tars.io/LocalVolumeGID: "0"
        tars.io/LocalVolumeMode: "755"
        tars.io/LocalVolumeUID: "1000"
    labels:
        tars.io/LocalVolume: remote-log-dir
        tars.io/ServerApp: tars
        tars.io/ServerName: tarslog
    name: remote-log-dir-tars-tarslog-0
    namespace: default
spec:
    resources:
        requests:
            storage: 1G
    selector:
        matchLabels:
        tars.io/LocalVolume: remote-log-dir
        tars.io/ServerApp: tars
        tars.io/ServerName: tarslog
    storageClassName: t-storage-class
    volumeMode: Filesystem
    volumeName: default-remote-log-dir-tars-tarslog-c4e31c6
```

### LocalPV对应pv

pv是由tars.tarsagent来展开生成, 并创建的, 并且会每台节点机都会创建出来(default-remote-log-dir-tars-tarslog-36520b71 名字每台节点机都不同), 当完成tars服务根据pvc完成pv绑定后, 就动态绑定到具体某一个pv了, 从而不再偏移. tars.tarsagent会定期删除一直没有绑定的pv.

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
annotations:
    pv.kubernetes.io/bound-by-controller: "yes"
    pv.kubernetes.io/provisioned-by: agent-downloader-provisioner-node8.67
labels:
    kubernetes.io/hostname: node8.67
    tars.io/LocalVolume: remote-log-dir
    tars.io/ServerApp: tars
    tars.io/ServerName: tarslog
name: default-remote-log-dir-tars-tarslog-36520b71
spec:
    accessModes:
    - ReadWriteOnce
    capacity:
        storage: 5Gi
    claimRef:
        apiVersion: v1
        kind: PersistentVolumeClaim
        name: remote-log-dir-tars-tarslog-1
        namespace: default
        resourceVersion: "42308569"
        uid: 869d9f9f-7313-49aa-b583-08a8fdce8668
    local:
        path: /usr/local/app/tars/host-mount/default/tars.tarslog/remote-log-dir
    nodeAffinity:
        required:
            nodeSelectorTerms:
            - matchExpressions:
                - key: kubernetes.io/hostname
                operator: In
                values:
                - node8.67
    persistentVolumeReclaimPolicy: Delete
    storageClassName: t-storage-class
    volumeMode: Filesystem
status:
    phase: Bound     
```

说明:
- 由于使用的是LocalPV, 因此这里存储容量其实没用, 只是占位符!
- ```/usr/local/app/tars/host-mount```实际是宿主机路径
- 实际pod中的存储路径, 会映射到宿主机的: /usr/local/app/taf/host-mount/${namespace}/$app.$server/${LocalPV Name}, 目录下


## 使用虚拟的 Delay-Bind LocalPV 达到延迟绑定效果

tserver 中定义了 HostIPC, HostNetwork, HostPorts时, 这些服务一般也不能漂移了, 因此这里也用到LocalPV的方式来实现这个技巧!

tarscontroller控制器会根据tserver, 并在生成的statefulset中添加虚拟的名为 "delay-bind" VolumeClainTemplates 项:

```go
    func BuildStatefulSetVolumeClainTemplates(tserver *crdV1beta1.TServer) []k8sCoreV1.PersistentVolumeClaim {
        var volumeClainTemplates []k8sCoreV1.PersistentVolumeClaim
        for _, mount := range tserver.Spec.K8S.Mounts {
            if mount.Source.PersistentVolumeClaimTemplate != nil {
                pvc := mount.Source.PersistentVolumeClaimTemplate.DeepCopy()
                pvc.Name = mount.Name
                volumeClainTemplates = append(volumeClainTemplates, *pvc)
            }
            if mount.Source.TLocalVolume != nil {
                volumeClainTemplates = append(volumeClainTemplates, *BuildTVolumeClainTemplates(tserver, mount.Name))
            }
        }

        //代码重点是这里!
        if tserver.Spec.K8S.HostIPC || tserver.Spec.K8S.HostNetwork || len(tserver.Spec.K8S.HostPorts) > 0 {
            volumeClainTemplates = append(volumeClainTemplates, *BuildTVolumeClainTemplates(tserver, THostBindPlaceholder))
        }

        return volumeClainTemplates
    }
```

从而相当于有一个LocalPV(虽然这个LocalPV不存储数据), 完成了服务和机器的绑定(不再漂移), tars服务自己的LocalPV就不能去delay-bind名字了!

展开的statefulset示例如下:
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
    name: tars-tarsconfig
spec:
    template:
        metadata:
            name: tars-tarsconfig
        spec:
            hostIPC: true
            containers:
            -   image: harbor.12345up.com/tars-helm/tars.tarsconfig:p7
                name: tars-tarsconfig
volumeClaimTemplates:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    labels:
        tars.io/LocalVolume: delay-bind
        tars.io/ServerApp: tars
        tars.io/ServerName: tarsconfig
    name: delay-bind
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
        requests:
            storage: 1G
    selector:
        matchLabels:
            tars.io/LocalVolume: delay-bind
            tars.io/ServerApp: tars
            tars.io/ServerName: tarsconfig
    storageClassName: t-storage-class
    volumeMode: Filesystem
```

