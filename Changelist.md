## v1.1.1 20211201

### en

- The docker image is uniformly switched to the official Debian image
- The helm.build.id in tarsconroller & tarsframework yaml is set to latest by default
- The tarsimage process is optimized and logs are added
- Fixed secretname in tserver CRD
- Fixed the bug that tarsnode does not exit when the master control is not started
- Fixed php template
- Update charts of tarsconroller & tarsframework 的
- Tarscpp & tarsweb dependency changed to submodule mode
- Utf8 character set is used inside the image

### cn

- docker 镜像统一切换到官方的 debian 镜像
- tarsconroller & tarsframework yaml 中的 helm.build.id 缺省设置为 latest
- 优化了 tarsimage 的过程, 增加了日志
- 修复了 TServer crd 中的 secretName
- 修复了主控没启动时, tarsnode 不退出的 bug
- 修复了 php 的模板
- 更新了 tarsconroller & tarsframework 的 charts
- tarscpp & tarsweb 依赖改成了 submodule 模式
- 镜像内部都采用 utf8 字符集

## v1.1.0 20211024

### en

- tarsweb>=v2.4.19, tarscpp>=v2.4.22 or >= 3.0.1
- tarslog support call train
- CRD update. Note that when upgrading tars, you need to upgrade tarscontroller and CRD

### cn

- tarsweb>=v2.4.19, tarscpp>=v2.4.22 or >= 3.0.1
- tarslog 支持调用链
- crd 更新, 注意升级 Tars 时候需要升级 tarscontroller 以及更新 CRD

## v1.0.0 20210911

### en

- In the first version, tarsweb>=v2.4.18, tarscpp>=v2.4.22 or >= 3.0.0

### cn

- 第一个版本, tarsweb 要求>=v2.4.18, tarscpp 要求>=v2.4.22 or >= 3.0.0
