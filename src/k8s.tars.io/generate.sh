#! /bin/bash

rm -rf client-go
"${PWD}"/hack/generate-groups.sh "all" "k8s.tars.io/client-go" "k8s.tars.io/apis" "tars:v1beta1,v1beta2,v1beta3" --output-base=".." --go-header-file="hack/hack/boilerplate.go.txt"
