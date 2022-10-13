package runner

import (
	"k8s.io/client-go/kubernetes"
	k8sClientCmd "k8s.io/client-go/tools/clientcmd"
)

func CreateK8SClient(masterUrl, kubeConfigPath string) (kubernetes.Interface, error) {
	clusterConfig, err := k8sClientCmd.BuildConfigFromFlags(masterUrl, kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(clusterConfig)
}
