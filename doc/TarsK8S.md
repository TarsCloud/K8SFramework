
# TarsK8S 使用的 k8s 特性

## tars 服务与 k8s workload

每个 tars 服务在 k8s 集群上对应一个 service 和 一个 statefulset ,name 都为lower($ServerApp)-lower($ServerName)

服务的 端口配置, 副本伸缩，版本变更等操作最终都是通过更新 service,statefulset 对应参数来完成的

此外,在 service 上和 statefulset 上分别添加了 annotations 和 label

+ service annotations and label

```yml
annotations:
  tars.io/Servant: '{"notifyobj":{"Name":"NotifyObj","Port":10000,"HostPort":0,"Threads":3,"Connections":10000,"Capacity":10000,"Timeout":60000,"Istars":true,"IsTcp":true}}'
labels:
  tars.io/ServerApp: tars
  tars.io/ServerName: tarsnotify
```

+ statefulset annotations and label

```yaml
  annotations:
    tars.io/NodeSelector: '{"Kind":"AbilityPool","Value":[]}'
    tars.io/NotStacked: "false"
  creationTimestamp: "2020-07-21T01:54:22Z"
  generation: 1
  labels:
    tars.io/ServerApp: tars
    tars.io/ServerName: tarsnotify
```

annotations 的含义下文会有阐述


## 节点选择

TarsK8S 提供了三种节点选择方式.

1. 指定节点(NodeBind)
   
   指定节点是限定服务所属Pod只能调度在指定节点上.
   
   该功能是通过定义 Statefulset.Spec.Template.Spec.Affinity.NodeAffinity 参数实现的,示例配置如下:

    ```json
    {
      "affinity": {
        "nodeAffinity": {
          "requiredDuringSchedulingIgnoredDuringExecution": {
            "nodeSelectorTerms": [
              {
                "matchExpressions": [
                  {
                    "key": "kubernetes.io/hostname",
                    "operator": "In",
                    "values": [
                      "kube.node117",
                      "kube.node118",
                      "kube.node119"
                    ]
                  }
                ]
              }
            ]
          }
        }
      }
    }
    ```

2. 能力筛选(AbilityPool)

    使用能力筛选功能分为两步:
    
    首先,规划好每个节点运行应用(ServerApp)的的能力

    ```text
    规划节点的能力(ability)就是给节点添加/删除 tars.io/ablility.($ServerApp) 标签的过程

    举例来说，
        要增加 kube.node117 ,kube.node118,kube.node119 运行 tars 应用的能力,
        就在 kube.node117 ,kube.node118,kube.node119 节点上添加 tars.io/ablility.tars 标签
        
        要取消 kube.node118,kube.node119 节点运行 Test 应用的能力,
        就在 kube.node118,kube.node119 节点上删除 tars.io/ablility.Test 标签


    每个节点可以有多种能力，互不冲突
    
    在Pod已经通过能力筛选方式调度到某个节点之后,取消节点的能力(ability)，不会影响当前Pod的运行
    
    ```

    然后,在部署服务时，节点选择方式选择为 能力筛选(AbilityPool)

    最终生成的 Statefulset.Spec.Template.Spec.Affinity.NodeAffinity 参数如下:

    ```json
    {
      "affinity": {
        "nodeAffinity": {
          "requiredDuringSchedulingIgnoredDuringExecution": {
            "nodeSelectorTerms": [
              {
                "matchExpressions": [
                  {
                    "key": "tars.io/ability.tars",
                    "operator": "Exists"
                  }
                ]
              }
            ]
          }
        }
      }
    }
    ```

3. 公共节点(PublicPool)

    使用公共节点功能分为两步:
    
    首先,规划好哪些个节点是公共节点
    
    ```text
    规划公共节点就是给节点添加/删除 tars.io/public 标签的过程

    举例来说，
        要增加 kube.node117 ,kube.node118,kube.node119 节点为公共节点,
        就在 kube.node117 ,kube.node118,kube.node119 节点上添加 tars.io/public 标签
        
        要取消 kube.node118,kube.node119 的公共节点属性
        就在 kube.node118,kube.node119 节点上删除 tars.io/public 标签

        在Pod已经通过公共节点的节点选择方式调度到某个节点之后,取消节点的公共属性，不会影响当前Pod的运行
    ```
   
   该功能是通过定义 Statefulset.Spec.Template.Spec.Affinity.NodeAffinity 参数实现的,示例配置如下

    ```json
    {
      "affinity": {
        "nodeAffinity": {
          "requiredDuringSchedulingIgnoredDuringExecution": {
            "nodeSelectorTerms": [
              {
                "matchExpressions": [
                  {
                    "key": "tars.io/public",
                    "operator": "Exists"
                  }
                ]
              }
            ]
          }
        }
      }
    }
    ```

## 服务暴露

在某些情况下,需要将服务的地址和端口直接对集群外暴露,TarsK8S提供了两种暴露方式,如非特殊需要,建议使用 HostPort 方式暴露

1. HostNetwork

   启用 HostNetwork 方式的步骤是 节点选择方式为 “指定节点”(NodeBind)->勾选 HostNetwork.
   此种方式会影响 Statefulset.Spec.Template.Spec.HostNetWork 属性 

2. HostPort

   启用 HostNetwork 方式的步骤是 节点选择方式为 “指定节点”(NodeBind)->勾选 HostPort->在 Servant选项中填写 每个 ServantPort 对应 ServantHostPort

   此种方式会影响 Statefulset.Spec.Template.Spec.Containers[0].Ports 属性,示例如下:
   
   ```json
    "addresses": [
        {
            "containerPort": 232,
            "hostPort": 9090,
            "name": "testobj",
            "protocol": "TCP"
        }
    ],

   ```
 HostNetwork 与 HostPort 方式只能二选一，在同时配置的情况下,只有 HostNetwork 生效

## 服务堆叠

在默认情况下，节点可以同时运行同一个服务的多个Pod,此种情况称为堆叠.在一些高可用场景下，可能希望Pod分散在不同的Node,可以使用 不允许堆叠 选项，来强制Pod分布在不同节点. (后续会添加分散条件，比如强制Pod分散在不同的可用区)

此功能是通过定义 Statefulset.Spec.Template.Spec.Affinity.PodAntiAffinity 实现的,示例如下:

```json
{
  "affinity": {
    "podAntiAffinity": {
      "requiredDuringSchedulingIgnoredDuringExecution": [
        {
          "labelSelector":{
            "tars.io/ServerApp" :"tars",
            "tars.io/ServerName":"tars-tarsnotify"
          },
          "namespaces":"tars",
          "topologyKey": "kubernetes.io/hostname"
        }
      ]
    }
  }
}
```

## HostIpc

某些服务程序可能会使用系统 System IPC 功能(共享内存，消息队列等),如果重建Pod会导致数据丢失，因此有必要让Pod 使用宿主机命名空间IPC资源，而不是Pod自身命名空间 IPC 资源.如果启用该选项，只要 Pod不被调度到另外的节点，IPC 内保存的数据不会丢失

启用 HostIpc 方式的步骤 节点选择方式为 “指定节点”(NodeBind)->勾选 HostIpc
此种方式会影响 Statefulset.Spec.Template.Spec.HostIpc 属性

为了确保 Pod不被调度到另外的节点，可以设置 不允许服务堆叠以及设置服务Pod数量严格等于指定的节点数


## ReadinessGates
  TarsK8S 使用了 k8s 的 ReadinessGates 特性，为每一个 Statefulset 配置了  
  ```json
  {
    "readinessGates":[
      {
        "conditionsType":"tars.io/service"
      }
    ]
  }
  ```
  当 tarsregistry 程序接收到 tarsnode 发送的 state 数据时
  + 如果 state!="Active" ,会将 pod["tars.io/service"].status 设置为 false . 这会使 k8s 自动从 endpoints 种摘除此 pod

  + 如果 state=="Active",会将 pod["tars.io/service"].status 设置为 true, 这会使 k8s 自动将此 pod 加入到 endpoints


# TarsK8S 的运维

在一般情况下，TarsK8S 的运维方式 与 tars 几乎一致或见名(界面)知意,此处只介绍不一致的地方

## 伸缩 tarsregistry, tarsimage , tarsadmin,tarsweb
 因为tarsregistry, tarsimage ,tarsadmin, tarsweb 与业务服务部署方式不太一致.在tarsweb上是无法管理这些服务的, 当需要伸缩时，可以使用 kubectl 修改。
 另外，tarsregistry, tarsadmin 承担着同步集群状态数据到 tars_db 的工作 ，需要确保无论何时，必须有一个 tarsregistry pod,tarsadmin 存在，否则会导致 tars_db 记录的数据与实际状态不一致.

## 修改 tars-db参数
secrets/tars-db 中存储了tars-db相关信息
如果需要修改相关值，则需要先修改 secrets/tars-db ,然后重启所有 tars-tarsregistry pod 使修改生效
后续可能会提供 tarsweb 提供管理接口

## 修改镜像参数
configmap/tars-image 中存储了镜像仓库地址，镜像仓库用户名，基础镜像Tag等信息. tars-tarsimage pod 在启动时会将 configmap/tars-image 挂载为环境变量，并在工作中使用

如果需要修改相关值，则需要先修改 configmap/tars-image ,然后重启所有 tars-tarimage pod
后续会升级为不需要重启 tars-tarsimage,延迟数秒后自动生效
后续可能会提供 tarsweb 提供管理接口 

此外,tarsimaeg 只支持单点运行,请保持  tarsimage pod数为 1个

## 更新 tarsregistry, tarsimage , tarsadmin, tarsweb 版本
 <!-- todo -->

## 规划节点的Ability
 
 通过 tars-tarsweb 提供的 能力界面完成

## 规划公共节点
  
 通过 tars-tarsweb 提供的 节点管理界面完成
