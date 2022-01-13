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
	"os/signal"
	"syscall"
	"time"
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

	var namespace string
	var bs []byte
	if bs, err = ioutil.ReadFile(namespaceFile); err != nil {
		utilRuntime.HandleError(fmt.Errorf("fatal error :  unable to load namespace value, %s", err.Error()))
		os.Exit(-1)
	} else {
		namespace = string(bs)
	}
	return &K8SContext{
		k8sClient: k8sClient,
		crdClient: crdClient,
		namespace: namespace,
	}
}

func main() {

	stopChan := make(chan struct{})
	k8sContext := CreateK8SContext("", "")

	watcher := NewWatcher(k8sContext)
	watcher.Start(stopChan)
	watcher.WaitSync(stopChan)

	//todo check config values;

	builder = NewBuilder(k8sContext)
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
	time.Sleep(time.Second * 1)
}
