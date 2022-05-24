package main

import (
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	k8sSchema "k8s.io/client-go/kubernetes/scheme"
	k8sClientCmd "k8s.io/client-go/tools/clientcmd"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
	crdScheme "k8s.tars.io/client-go/clientset/versioned/scheme"
)

type K8SContext struct {
	k8sClient kubernetes.Interface
	crdClient crdVersioned.Interface
	namespace string
}

func CreateK8SContext(masterUrl, kubeConfigPath string) (*K8SContext, error) {
	clusterConfig, err := k8sClientCmd.BuildConfigFromFlags(masterUrl, kubeConfigPath)
	if err != nil {
		return nil, err
	}

	k8sClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, err
	}

	crdClient, err := crdVersioned.NewForConfig(clusterConfig)
	if err != nil {
		return nil, err
	}

	err = crdScheme.AddToScheme(k8sSchema.Scheme)
	if err != nil {
		return nil, err
	}

	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	var bs []byte
	if bs, err = ioutil.ReadFile(namespaceFile); err != nil {
		if err != nil {
			return nil, err
		}
	}

	return &K8SContext{
		k8sClient: k8sClient,
		crdClient: crdClient,
		namespace: string(bs),
	}, nil
}
