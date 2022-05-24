# 说明

该目录主要提供 CICD 相关的脚本和镜像

- base-compiler.Dockerfile: 制作编译以及发布镜像, 给业务服务使用
- tools: CICD 流程中需要的 helm 包, 发布脚本等, 会内置到 base-compiler 中
