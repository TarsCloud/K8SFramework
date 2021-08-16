# 关于

- TarsK8S 是为了将 Tars 部署在 K8S平台上而做的适应性改造项目.项目已初具雏形,但还有很多细节和功能需要完善.

- TarsK8S 改造的原则:
  1. 改造仅限于 framework 程序（tarsregistry ,tarsnode , tarsnotify , tarsconfig）
  2. 改造后的 framework 程序对业务服务暴露接口保持完全兼容
 
- 基于以上原则,现在的 tars服务程序在无需大改的情况下就可以迁移到 TarsK8S平台, 具体操作请参看《tars服务迁移说明》 

- TarsK8S 暂时不支持 Set ,但最终会支持

- TarsK8S 暂时不能部署 DCache 服务,但最终会支持

# TarsK8S 的组成

+ tarscontroller
    > tars 框架在 k8s 平台的控制器程序        
+ tarsregistry
    > k8s tarsRegistry 与 tars 框架中的　tarsRegistry　提供相同的业务服务接口,具备相同的功能. 在实现上是通过 k8s list-watch 机制来感知集群内的各个服务状态,并对外提供服务

+ tarsweb
    > k8s tarsWeb 是对外提供的管理界面

+ tarsadmin
    > k8s tarsAdmin 与 tars 中的　tarsAdminRegistry 服务功能类似, 是 tarsWeb 操作集群的中间层.

+ tarsnotify
    > k8s tarsNotify 与 tars 中的 tarsNotify 服务有相同的业务服务接口,在实现上有细微的差别,但是对业务服务透明.
              
+ tarsconfig
    > k8s tarsConfig 与 tars 中的 tarsConfig 服务有相同的业务服务接口.在使用上有细微差别,具体参见 《TarsK8S的使用细节》
                        
+ tarsproprety ,tarsstat, tarsqueryproperty ,tarsquerystat,tarslog
    > tarsProperty ,tarsStat, tarsQueryProperty ,tarsQueryStat,tarsLog 与 tars 中的同名服务使用相同代码构建，完全兼容.

+ tarsagent
  > tarsAgent 是 tarsk8S 新增的框架服务. 用于提供主机端口检测,日志清理等辅助功能.
 
+ tarsnode
    > k8st tarsnode 程序经过轻量化改造，集成在每一个业务服务镜像中. 作为容器内的1号进程运行
            
+ tarsimage
    > 此服务程序是 tars-k8s  框架新增的服务，用于提供镜像生成功能.

+ 删除了 tarspatch , tarsAdminRegistry

# TarsK8S 安装
  TarsK8S 以 helm 包的形式对外发布.每个 Helm 包包含了 完成的框架服务. 
  在安装时,以k8s命名空间为分割,每个命名空间可以且只可以部署一套框架服务,每套框架服务都有完整的功能组件,与其他命名空间的框架互相独立

 ```shell script
   helm install -n [namesace] --create-namesace tars-framwork -f value.yaml   
 ```

# Todo List
+ 增加用户登陆,用户管理,用户权限功能
+ 增加更多的 k8s 选项参数 例如节点,调度策略,存储挂载,资源限制等
+ 完成 tarscontroller 服务的高可用改造,支持 crd,api 版本升级
+ 完善的增强 tarsweb功能
