package common

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var dev  = flag.Bool("dev", false, "bool类型参数: 本地启动")
var config = flag.String("config", "/root/.kube/config", "string类型参数：本地启动时，配置文件路径")
var namespace = flag.String("namespace", "tars", "string类型参数：本地启动时，K8S命名空间")

func LoadEnv() (string, *rest.Config, error) {
	glog.Infof("load controller dev: %t, conf: %s\n", *dev, *config)

	var k8sNamespace string
	var k8sConfig *rest.Config
	var err error

	if !*dev {
		k8sNamespace, k8sConfig, err = loadK8S()
		if err != nil {
			return "", nil, fmt.Errorf("Load K8S Error: %s\n", err.Error())
		}
	} else {
		k8sNamespace, k8sConfig, err = loadK8SDev(*config, *namespace)
		if err != nil {
			return "", nil, fmt.Errorf("Load K8SDev Error: %s\n", err.Error())
		}
	}

	return k8sNamespace, k8sConfig, nil
}

func loadK8SDev(confPath, namespace string) (string, *rest.Config, error) {
	var k8sNamespace = namespace

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", confPath)
	if err != nil {
		return "", nil, fmt.Errorf("Get K8SConfig Value Error , Did You Run Program In K8S ? ")
	}
	return k8sNamespace, k8sConfig, nil
}

func loadK8S() (string, *rest.Config, error) {
	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	var k8sNamespace string

	if bs, err := ioutil.ReadFile(namespaceFile); err != nil {
		return "", nil, fmt.Errorf("Get K8SNamespace Value Error , Did You Run Program In K8S ? ")
	} else {
		k8sNamespace = string(bs)
	}

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return "", nil, fmt.Errorf("Get K8SConfig Value Error , Did You Run Program In K8S ? ")
	}

	return k8sNamespace, k8sConfig, nil
}
