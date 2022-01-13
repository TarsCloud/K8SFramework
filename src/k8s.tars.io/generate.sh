#! /bin/bash

rm -rf client-go
cd hack
./generate-groups.sh all k8s.tars.io/client-go k8s.tars.io/api "crd:v1beta1,v1beta2" --output-base="../.." --go-header-file="hack/boilerplate.go.txt"

