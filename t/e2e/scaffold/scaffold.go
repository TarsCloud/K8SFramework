package scaffold

import (
	"context"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sClientCmd "k8s.io/client-go/tools/clientcmd"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

type Options struct {
	Name      string
	K8SConfig string
}

type Scaffold struct {
	Opts      *Options
	K8SClient kubernetes.Interface
	CRDClient crdVersioned.Interface
	Namespace string
	t         testing.TestingT
}

func GetK8SConfigFile() string {
	configFile := os.Getenv("KUBECONFIG")
	if configFile == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		configFile = filepath.Join(u.HomeDir, ".kube", "config")
		if _, err := os.Stat(configFile); err != nil && !os.IsNotExist(err) {
			configFile = ""
		}
	}
	return configFile
}

func NewScaffold(o *Options) *Scaffold {
	defer ginkgo.GinkgoRecover()
	var s = &Scaffold{
		Opts: o,
	}
	clusterConfig, err := k8sClientCmd.BuildConfigFromFlags("", GetK8SConfigFile())

	if err != nil {
		return nil
	}

	s.K8SClient = kubernetes.NewForConfigOrDie(clusterConfig)

	s.CRDClient = crdVersioned.NewForConfigOrDie(clusterConfig)

	ginkgo.BeforeEach(s.beforeEach)
	ginkgo.AfterEach(s.afterEach)
	return s
}

func (s *Scaffold) beforeEach() {
	s.Namespace = fmt.Sprintf("tars-e2e-test-%s-%d", s.Opts.Name, time.Now().Nanosecond())
	namespace := &k8sCoreV1.Namespace{
		TypeMeta: k8sMetaV1.TypeMeta{},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: s.Namespace,
		},
		Spec:   k8sCoreV1.NamespaceSpec{},
		Status: k8sCoreV1.NamespaceStatus{},
	}
	_, err := s.K8SClient.CoreV1().Namespaces().Create(context.TODO(), namespace, k8sMetaV1.CreateOptions{})
	if err != nil {
		return
	}
}

func (s *Scaffold) afterEach() {
	err := s.K8SClient.CoreV1().Namespaces().Delete(context.TODO(), s.Namespace, k8sMetaV1.DeleteOptions{})
	assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", s.Namespace)
	time.Sleep(3 * time.Second)
}
