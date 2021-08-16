package main

import (
	"fmt"
	"io/ioutil"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	k8sSchema "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/clientcmd"
	crdClient "k8s.tars.io/client-go/clientset/versioned"
	crdScheme "k8s.tars.io/client-go/clientset/versioned/scheme"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type K8SOption struct {
	k8sClientInterface kubernetes.Interface
	crdClientInterface crdClient.Interface
	namespace          string
}

func LoadK8SOption() *K8SOption {
	//clusterConfig, err := clientcmd.BuildConfigFromFlags("", "/root/.kube/config")
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("fatal eror : %s", err.Error()))
		os.Exit(-1)
	}

	k8sClientInterface := kubernetes.NewForConfigOrDie(clusterConfig)
	crdClientInterface := crdClient.NewForConfigOrDie(clusterConfig)
	utilRuntime.Must(crdScheme.AddToScheme(k8sSchema.Scheme))

	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	var namespace string
	if bs, err := ioutil.ReadFile(namespaceFile); err != nil {
		utilRuntime.HandleError(fmt.Errorf("fatal error :  unable to load namespace value, %s", err.Error()))
		os.Exit(-1)
	} else {
		namespace = string(bs)
	}
	//namespace = "default"
	return &K8SOption{
		k8sClientInterface: k8sClientInterface,
		crdClientInterface: crdClientInterface,
		namespace:          namespace,
	}
}

func main() {

	stopChan := make(chan struct{})

	builder = NewBuildWorker()
	builder.Start(stopChan, MaximumConcurrencyBuildTask)

	restfulServer := NewRestfulServer()
	restfulServer.Start(stopChan)

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGCHLD, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for true {
		sig := <-sigChan
		switch sig {
		case syscall.SIGCHLD:
			var waitStatus syscall.WaitStatus
			_, _ = syscall.Wait4(-1, &waitStatus, syscall.WNOHANG, nil)
		default:
			break
		}
	}
	close(stopChan)
	time.Sleep(time.Second * 5)
}
