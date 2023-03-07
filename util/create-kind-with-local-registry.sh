#!/bin/sh
set -o errexit

cluster_name=$1
reg_name=$2
reg_port=$3
node_image=$4

if [ -z "${node_image}" ]; then
  #  node_image="kindest/node:v1.18.20"
  #  node_image="kindest/node:v1.19.16"
  #  node_image="kindest/node:v1.20.15"
  #  node_image="kindest/node:v1.21.14"
  #  node_image="kindest/node:v1.22.15"
  #  node_image="kindest/node:v1.23.13"
  node_image="kindest/node:v1.24.7"
fi

cat <<EOF | kind create cluster --name "${cluster_name}" --image ${node_image} --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:5000"]
EOF

# connect the registry to the cluster network if not already connected
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${reg_name}")" = 'null' ]; then
  docker network connect "kind" "${reg_name}"
fi

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
