package v1beta3

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1Beta3 "k8s.tars.io/apis/tars/v1beta3"
	crdMetaTools "k8s.tars.io/meta"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"time"
)

var _ = ginkgo.Describe("try create tars server and check filed", func() {
	opts := &scaffold.Options{
		Name:     "default",
		SyncTime: 800 * time.Millisecond,
	}

	s := scaffold.NewScaffold(opts)

	var tsLayout *tarsV1Beta3.TServer
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

		tsLayout = &tarsV1Beta3.TServer{
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

	ginkgo.It("check filed value", func() {
		tserver, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tserver)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:  App,
			tarsMeta.TServerNameLabel: Server,
			tarsMeta.TTemplateLabel:   Template,
			tarsMeta.TSubTypeLabel:    string(tarsV1Beta3.TARS),
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tserver.Labels))
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.Important)
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.Tars)
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.Tars.Servants)
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.Tars.Template)
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.Tars.AsyncThread)
		assert.Nil(ginkgo.GinkgoT(), tserver.Spec.Tars.Ports)
		assert.Nil(ginkgo.GinkgoT(), tserver.Spec.Normal)
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.K8S)
		assert.False(ginkgo.GinkgoT(), tserver.Spec.K8S.HostIPC)
		assert.False(ginkgo.GinkgoT(), tserver.Spec.K8S.HostNetwork)
		assert.Nil(ginkgo.GinkgoT(), tserver.Spec.K8S.HostPorts)
		assert.False(ginkgo.GinkgoT(), tserver.Spec.K8S.DaemonSet)
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.K8S.Replicas)
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.K8S.NodeSelector)
		assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.K8S.ImagePullPolicy)

		expectedReadinessGates := []string{tarsMeta.TPodReadinessGate}
		assert.Equal(ginkgo.GinkgoT(), expectedReadinessGates, tserver.Spec.K8S.ReadinessGates)
	})

	ginkgo.It("try remove immutable filed", func() {
		removeFields := map[string]interface{}{
			"/spec/app":     nil,
			"/spec/server":  nil,
			"/spec/subType": nil,
			"/spec/tars":    nil,
			"/spec/k8s":     nil,
		}
		for k := range removeFields {
			jsonPath := crdMetaTools.JsonPatch{
				{
					OP:   crdMetaTools.JsonPatchRemove,
					Path: k,
				},
			}
			bs, _ := json.Marshal(jsonPath)
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		}
	})

	ginkgo.It("try update immutable filed", func() {
		updateFields := map[string]string{
			"/spec/app":     scaffold.RandStringRunes(3),
			"/spec/server":  scaffold.RandStringRunes(5),
			"/spec/subType": scaffold.RandStringRunes(5),
		}
		for k, v := range updateFields {
			jsonPath := crdMetaTools.JsonPatch{
				{
					OP:    crdMetaTools.JsonPatchReplace,
					Path:  k,
					Value: v,
				},
			}
			bs, _ := json.Marshal(jsonPath)
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		}
	})

	ginkgo.It("try update subType", func() {
		jsonPath := crdMetaTools.JsonPatch{
			{
				OP:    crdMetaTools.JsonPatchReplace,
				Path:  "/spec/subType",
				Value: tarsV1Beta3.Normal,
			},
			{
				OP:   crdMetaTools.JsonPatchRemove,
				Path: "/spec/tars",
			},
			{
				OP:   crdMetaTools.JsonPatchAdd,
				Path: "/spec/normal",
				Value: &tarsV1Beta3.TServerNormal{
					Ports: []*tarsV1Beta3.TServerPort{},
				},
			},
		}
		bs, _ := json.Marshal(jsonPath)
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})
})
