# helm包管理

管理基础的TAF服务的helm按转包模板, 并生成charts包.

## 目录说明

- tars-server 

tars服务安装包模板, 可以生成tgz包, 以供业务方使用. 

如何更新:
- 每次修改安装template以后都要修改Chart.yaml里面的版本号(version)
- 修改docs/helm-template/Charts.yaml 里面的依赖的版本dependencies.version
- 提交到git, gitlab会自动编译charts包并推送到harbor仓库

- charts: 

重新生成tgz包:
```
sh create.sh
```


