
# 关于存储

## 支持 LocalPV ，并可以在 Web 上配置
   
- 我们的大部分使用场景是自建集群,没有使用网络盘的条件, 因此,支持并且可以简单配置 LocalPV 非常重要

## 在 配置 Hostnetwork ,HostIPC ,HostPort 后，Pod 与 节点延迟 绑定

```
    在当前的版本中(v1beta1),使用 Nodebind 方式实现 节点绑定的效果 ,但该方式存在严重缺陷
```

3. 在 TServer 中，除 LabelMatch 由用户主动输入节点名外，不与集群的特定信息关联.

```
   TServer 与集群特定信息关联后，非常影响移植性.
```

4. 融合亲和性与LabelMatch
```
  在小集群和客户集群中，亲和性功能显得比较多余
  同时, 亲和性与 LabelMath ,NodeBind 无法共存. 有特殊需求时,使用方式违反直觉
```

5. 支持未来的 某个业务服务的特殊需求

```
  比如，业务服务需求节点由特定在资源 ，比如 SSD, 公网，GPU
```



# 升级方式

## 1. 增加 tsever.k8s.mounts.source.TLocalVolume 项

schema:
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
sample:
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
pvc:
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

pv:  -> tarsagent 建立， k8s 控制器 执行 bound, release 操作

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

## 2. 使用虚拟的 Delay-Bind LocalPV 达到延迟绑定效果

控制器发现 tserver 中定义了 HostIPC, HostNetwork, HostPorts.
则会在 生成的 statefulset 中 添加  虚拟的名为 "dealy-bind" VolumeClainTemplates 项

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

        if tserver.Spec.K8S.HostIPC || tserver.Spec.K8S.HostNetwork || len(tserver.Spec.K8S.HostPorts) > 0 {
            volumeClainTemplates = append(volumeClainTemplates, *BuildTVolumeClainTemplates(tserver, THostBindPlaceholder))
        }

        return volumeClainTemplates
    }
```

statefulset:
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


## 3. 调整亲和性的逻辑

### 当前的亲和性逻辑的问题
- AbilityPool 和 LabelMath |Nodebind 互斥，当业务服务有特殊需求时,会选择
  LabelMath, NodeBind，使得亲和性逻辑失效,造成“惊讶”.

- 在小规模集群，亲和性不具备使用意义，反而对业务部署造成干扰.


### 升级后的亲和性使用方式

- App 级别 节点标签不变: tars.io/ability.${App} , 新增 Server级标签  tars.io/ability.${App}.${Server}

- 添加独立字段表示亲和性选项

```yaml
k8s:
  type: object
  properties:
    abilityAffinity:
      type: string
      enum: [ AppRequired,ServerRequired, AppOrServerPreferred,None ]
      default: AppOrServerPreferred
```


释义：

+ AppRequired：            在满足其他条件后，节点必须有 tars.io/ability.${App} 标签

+ ServerRequired           在满足其他条件后，节点必须有 tars.io/ability.${App}.${Server} 标签

+ AppOrServerPreferred:    在满足其他条件后，优先选择有 tars.io/ability.${App}.${Server}, tars.io/ability.${App} 标签的节点. 如果节点 没有 ability 标签， 则忽略

+ None： 不对节点标签做要求

