package v1beta2

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
	tarsCrdV1Beta2 "k8s.tars.io/crd/v1beta2"
	tarsMetaTools "k8s.tars.io/meta/tools"
	tarsMetaV1Beta2 "k8s.tars.io/meta/v1beta2"
	"strings"
	"time"
)

var _ = ginkgo.Describe("try create/update tars server and check service", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  1500 * time.Millisecond,
	}

	s := scaffold.NewScaffold(opts)

	var Resource = "test-testserver"
	var App = "Test"
	var Server = "TestServer"
	var Template = "tt.cpp"
	var FirstObj = "FirstObj"
	var SecondObj = "SecondObj"

	ginkgo.BeforeEach(func() {
		ttLayout := &tarsCrdV1Beta2.TTemplate{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      Template,
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta2.TTemplateSpec{
				Content: "tt.cpp content",
				Parent:  Template,
			},
		}
		_, err := s.CRDClient.CrdV1beta2().TTemplates(s.Namespace).Create(context.TODO(), ttLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)
		tsLayout := &tarsCrdV1Beta2.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      Resource,
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta2.TServerSpec{
				App:       App,
				Server:    Server,
				SubType:   tarsCrdV1Beta2.TARS,
				Important: 5,
				Tars: &tarsCrdV1Beta2.TServerTars{
					Template:    Template,
					Profile:     "",
					AsyncThread: 3,
					Servants: []*tarsCrdV1Beta2.TServerServant{
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
				K8S: tarsCrdV1Beta2.TServerK8S{
					AbilityAffinity: tarsCrdV1Beta2.None,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					ImagePullPolicy: k8sCoreV1.PullAlways,
					LauncherType:    tarsCrdV1Beta2.Background,
				},
			},
		}
		_, err = s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)
	})

	ginkgo.It("before update", func() {
		service, err := s.K8SClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), service)

		expectedLabels := map[string]string{
			tarsMetaV1Beta2.TServerAppLabel:  App,
			tarsMetaV1Beta2.TServerNameLabel: Server,
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
		jsonPatch := tarsMetaTools.JsonPatch{
			{
				OP:   tarsMetaTools.JsonPatchAdd,
				Path: "/spec/tars/servants/0",
				Value: &tarsCrdV1Beta2.TServerServant{
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
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		service, err := s.K8SClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), service)

		expectedLabels := map[string]string{
			tarsMetaV1Beta2.TServerAppLabel:  App,
			tarsMetaV1Beta2.TServerNameLabel: Server,
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
		jsonPatch := tarsMetaTools.JsonPatch{
			{
				OP:   tarsMetaTools.JsonPatchReplace,
				Path: "/spec/tars/servants/1",
				Value: &tarsCrdV1Beta2.TServerServant{
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
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		service, err := s.K8SClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), service)

		expectedLabels := map[string]string{
			tarsMetaV1Beta2.TServerAppLabel:  App,
			tarsMetaV1Beta2.TServerNameLabel: Server,
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
		jsonPatch := tarsMetaTools.JsonPatch{
			{
				OP:   tarsMetaTools.JsonPatchRemove,
				Path: "/spec/tars/servants/1",
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		service, err := s.K8SClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), service)

		expectedLabels := map[string]string{
			tarsMetaV1Beta2.TServerAppLabel:  App,
			tarsMetaV1Beta2.TServerNameLabel: Server,
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
		jsonPatch := tarsMetaTools.JsonPatch{
			{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/spec/k8s/daemonSet",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		_, err = s.K8SClient.CoreV1().Services(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), errors.IsNotFound(err))
	})
})
