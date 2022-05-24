按照如下操作生成client-go代码, 进入 k8s.tars.io

```shell
 cd hack
 ./generate-groups.sh all k8s.tars.io/client-go k8s.tars.io/api "core:v1beta1,v1beta2,v1beta3" --output-base="../.." --go-header-file="hack/boilerplate.go.txt"
```
