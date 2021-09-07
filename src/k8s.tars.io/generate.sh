#! /bin/bash

rm -rf client-go
cd hack
./generate-groups.sh all k8s.tars.io/client-go k8s.tars.io/api "crd:v1alpha1,v1beta1" --output-base="../.." --go-header-file="hack/boilerplate.go.txt"

