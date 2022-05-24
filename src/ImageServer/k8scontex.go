package main

import (
	"fmt"
	"io/ioutil"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	k8sSchema "k8s.io/client-go/kubernetes/scheme"
	k8sClientCmd "k8s.io/client-go/tools/clientcmd"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
	crdScheme "k8s.tars.io/client-go/clientset/versioned/scheme"
	"os"
)

type K8SContext struct {
	k8sClient kubernetes.Interface
	crdClient crdVersioned.Interface
	namespace string
}

func CreateK8SContext(masterUrl, kubeConfigPath string) *K8SContext {
	clusterConfig, err := k8sClientCmd.BuildConfigFromFlags(masterUrl, kubeConfigPath)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("fatal eror : %s", err.Error()))
		os.Exit(-1)
	}

	k8sClient := kubernetes.NewForConfigOrDie(clusterConfig)

	crdClient := crdVersioned.NewForConfigOrDie(clusterConfig)

	utilRuntime.Must(crdScheme.AddToScheme(k8sSchema.Scheme))

	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	var bs []byte
	if bs, err = ioutil.ReadFile(namespaceFile); err != nil {
		utilRuntime.HandleError(fmt.Errorf("fatal error :  unable to load namespace value, %s", err.Error()))
		os.Exit(-1)
	}

	return &K8SContext{
		k8sClient: k8sClient,
		crdClient: crdClient,
		namespace: string(bs),
	}
}
