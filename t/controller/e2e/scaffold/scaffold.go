package scaffold

import (
	"context"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsRuntime "k8s.tars.io/runtime"
	"time"
)

type Options struct {
	Name     string
	SyncTime time.Duration
}

type Scaffold struct {
	Opts      *Options
	Namespace string
	t         testing.TestingT
}

func NewScaffold(o *Options) *Scaffold {
	defer ginkgo.GinkgoRecover()
	var s = &Scaffold{
		Opts: o,
	}

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
	_, err := tarsRuntime.Clients.K8sClient.CoreV1().Namespaces().Create(context.TODO(), namespace, k8sMetaV1.CreateOptions{})
	if err != nil {
		return
	}
}

func (s *Scaffold) afterEach() {
	err := tarsRuntime.Clients.K8sClient.CoreV1().Namespaces().Delete(context.TODO(), s.Namespace, k8sMetaV1.DeleteOptions{})
	assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", s.Namespace)
}
