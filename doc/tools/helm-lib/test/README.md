
# 本地测试tars-server charts

```
helm dependency update helm-test
helm install helm-test -n tars-dev --dry-run helm-test
helm template helm-test -n tars-dev --dry-run helm-test --debug
```
