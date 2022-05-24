
# 本地测试tars-server charts

```
helm dependency update helm-test
helm install helm-test -n tars-dev --dry-run helm-test
helm install helm-test -n tars-dev -f helm-test/values-no-label-match.yaml --dry-run helm-test
helm install helm-test -n tars-dev -f helm-test/values-no-config.yaml --dry-run helm-test
helm install helm-test -n tars-dev -f helm-test/values-no-mounts.yaml --dry-run helm-test
helm template helm-test -n tars-dev --dry-run helm-test --debug
```
