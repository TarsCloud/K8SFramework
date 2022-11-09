package v1beta3

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1Beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"

	"strings"
	"time"
)

var _ = ginkgo.Describe("try create/update tars server and check service", func() {
	opts := &scaffold.Options{
		Name:     "default",
		SyncTime: 800 * time.Millisecond,
	}

	s := scaffold.NewScaffold(opts)

	var Resource = "test-testserver"
	var App = "Test"
	var Server = "TestServer"
	var Template = "tt.cpp"
	var FirstObj = "FirstObj"
	var SecondObj = "SecondObj"

	ginkgo.BeforeEach(func() {
		ttLayout := &tarsV1Beta3.TTemplate{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      Template,
				Namespace: s.Namespace,
			},
			Spec: tarsV1Beta3.TTemplateSpec{
				Content: "tt.cpp content",
				Parent:  Template,
			},
		}
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TTemplates(s.Namespace).Create(context.TODO(), ttLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		tsLayout := &tarsV1Beta3.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      Resource,
				Namespace: s.Namespace,
			},
			Spec: tarsV1Beta3.TServerSpec{
				App:       App,
				Server:    Server,
				SubType:   tarsV1Beta3.TARS,
				Important: 5,
				Tars: &tarsV1Beta3.TServerTars{
					Template:    Template,
					Profile:     "",
					AsyncThread: 3,
					Servants: []*tarsV1Beta3.TServerServant{
						{
							Name:       FirstObj,
							Port:       10000,
							Thread:     3,
							Connection: 1000,
							Capacity:   1000,
							Timeout:    1000,
							IsTars:     true,
							IsTcp:      true,
						},
						{
							Name:       SecondObj,
							Port:       10001,
							Thread:     3,
							Connection: 1000,
							Capacity:   1000,
							Timeout:    1000,
							IsTars:     true,
							IsTcp:      true,
						},
					},
				},
				K8S: tarsV1Beta3.TServerK8S{
					AbilityAffinity: tarsV1Beta3.None,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					ImagePullPolicy: k8sCoreV1.PullAlways,
					LauncherType:    tarsMeta.Background,
				},
			},
		}
		_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)
	})

	ginkgo.AfterEach(func() {
		_ = tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(s.Namespace).Delete(context.TODO(), Resource, k8sMetaV1.DeleteOptions{})
	})

	ginkgo.It("before update", func() {
		service, err := tarsRuntime.Clients.K8sClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), service)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:  App,
			tarsMeta.TServerNameLabel: Server,
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, service.Labels))
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, service.Spec.Selector))
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ServiceTypeClusterIP, service.Spec.Type)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ClusterIPNone, service.Spec.ClusterIP)
		assert.Equal(ginkgo.GinkgoT(), 2, len(service.Spec.Ports))

		var p0, p1 *k8sCoreV1.ServicePort
		for i := range service.Spec.Ports {
			if service.Spec.Ports[i].Name == strings.ToLower(FirstObj) {
				p0 = &service.Spec.Ports[i]
				continue
			}

			if service.Spec.Ports[i].Name == strings.ToLower(SecondObj) {
				p1 = &service.Spec.Ports[i]
				continue
			}
			assert.True(ginkgo.GinkgoT(), false, "unexpected port name")
		}
		assert.Equal(ginkgo.GinkgoT(), int32(10000), p0.Port)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ProtocolTCP, p0.Protocol)

		assert.Equal(ginkgo.GinkgoT(), int32(10001), p1.Port)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ProtocolTCP, p1.Protocol)
	})

	ginkgo.It("add servant", func() {
		var ThirdObj = "ThirdObj"
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchAdd,
				Path: "/spec/tars/servants/0",
				Value: &tarsV1Beta3.TServerServant{
					Name:       ThirdObj,
					Port:       10002,
					Thread:     3,
					Connection: 10000,
					Capacity:   10000,
					Timeout:    10000,
					IsTars:     true,
					IsTcp:      true,
				},
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		service, err := tarsRuntime.Clients.K8sClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), service)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:  App,
			tarsMeta.TServerNameLabel: Server,
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, service.Labels))
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, service.Spec.Selector))
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ServiceTypeClusterIP, service.Spec.Type)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ClusterIPNone, service.Spec.ClusterIP)
		assert.Equal(ginkgo.GinkgoT(), 3, len(service.Spec.Ports))

		var p0, p1, p2 *k8sCoreV1.ServicePort
		for i := range service.Spec.Ports {
			if service.Spec.Ports[i].Name == strings.ToLower(FirstObj) {
				p0 = &service.Spec.Ports[i]
				continue
			}

			if service.Spec.Ports[i].Name == strings.ToLower(SecondObj) {
				p1 = &service.Spec.Ports[i]
				continue
			}

			if service.Spec.Ports[i].Name == strings.ToLower(ThirdObj) {
				p2 = &service.Spec.Ports[i]
				continue
			}

			assert.True(ginkgo.GinkgoT(), false, "unexpected port name")
		}
		assert.Equal(ginkgo.GinkgoT(), int32(10000), p0.Port)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ProtocolTCP, p0.Protocol)

		assert.Equal(ginkgo.GinkgoT(), int32(10001), p1.Port)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ProtocolTCP, p1.Protocol)

		assert.Equal(ginkgo.GinkgoT(), int32(10002), p2.Port)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ProtocolTCP, p2.Protocol)
	})

	ginkgo.It("update servant", func() {
		var ThirdObj = "ThirdObj"
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchReplace,
				Path: "/spec/tars/servants/1",
				Value: &tarsV1Beta3.TServerServant{
					Name:       ThirdObj,
					Port:       10002,
					Thread:     3,
					Connection: 10000,
					Capacity:   10000,
					Timeout:    10000,
					IsTars:     true,
					IsTcp:      true,
				},
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		service, err := tarsRuntime.Clients.K8sClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), service)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:  App,
			tarsMeta.TServerNameLabel: Server,
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, service.Labels))
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, service.Spec.Selector))
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ServiceTypeClusterIP, service.Spec.Type)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ClusterIPNone, service.Spec.ClusterIP)
		assert.Equal(ginkgo.GinkgoT(), 2, len(service.Spec.Ports))

		var p0, p2 *k8sCoreV1.ServicePort
		for i := range service.Spec.Ports {
			if service.Spec.Ports[i].Name == strings.ToLower(FirstObj) {
				p0 = &service.Spec.Ports[i]
				continue
			}

			if service.Spec.Ports[i].Name == strings.ToLower(ThirdObj) {
				p2 = &service.Spec.Ports[i]
				continue
			}
			assert.True(ginkgo.GinkgoT(), false, "unexpected port name")
		}
		assert.Equal(ginkgo.GinkgoT(), int32(10000), p0.Port)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ProtocolTCP, p0.Protocol)

		assert.Equal(ginkgo.GinkgoT(), int32(10002), p2.Port)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ProtocolTCP, p2.Protocol)
	})

	ginkgo.It("delete servant", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchRemove,
				Path: "/spec/tars/servants/1",
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		service, err := tarsRuntime.Clients.K8sClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), service)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:  App,
			tarsMeta.TServerNameLabel: Server,
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, service.Labels))
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, service.Spec.Selector))
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ServiceTypeClusterIP, service.Spec.Type)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ClusterIPNone, service.Spec.ClusterIP)
		assert.Equal(ginkgo.GinkgoT(), 1, len(service.Spec.Ports))

		var p0 *k8sCoreV1.ServicePort
		for i := range service.Spec.Ports {
			if service.Spec.Ports[i].Name == strings.ToLower(FirstObj) {
				p0 = &service.Spec.Ports[i]
				continue
			}
			assert.True(ginkgo.GinkgoT(), false, "unexpected port name")
		}
		assert.Equal(ginkgo.GinkgoT(), int32(10000), p0.Port)
		assert.Equal(ginkgo.GinkgoT(), k8sCoreV1.ProtocolTCP, p0.Protocol)
	})

	ginkgo.It("daemonset", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchAdd,
				Path:  "/spec/k8s/daemonSet",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		_, err = tarsRuntime.Clients.K8sClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), errors.IsNotFound(err))
	})
})
