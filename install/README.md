# 安装Tars到K8S的helm包

为了方便使用者快速安装Tars到K8S上, 提供了安装tars框架的helm包, 这样不需要自己编译源码来完成安装.

主要说明:
- github启用了pages模式, 可以提供静态文件
- 运行./helm_index.sh, 会创建helm包(tarscontroller & tarsframework), 并创建在charts目录下
- helm_index.sh生成的(tarscontroller & tarsframework)的tgz包(charts目录下), 需要提交到github上

**注意: 更新tarscontroller & tarsframework 需要变变更它的版本号, 重新生成并提交**

