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
	tarsCrdV1Beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"

	"strings"
	"time"
)

var _ = ginkgo.Describe("try update tars server and check filed", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  800 * time.Millisecond,
	}

	s := scaffold.NewScaffold(opts)

	var tsLayout *tarsCrdV1Beta3.TServer
	var Resource = "test-testserver"
	var App = "Test"
	var Server = "TestServer"
	var Template = "tt.cpp"
	var FirstObj = "FirstObj"
	var SecondObj = "SecondObj"

	ginkgo.BeforeEach(func() {
		ttLayout := &tarsCrdV1Beta3.TTemplate{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      Template,
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta3.TTemplateSpec{
				Content: "tt.cpp content",
				Parent:  Template,
			},
		}
		_, err := s.CRDClient.CrdV1beta3().TTemplates(s.Namespace).Create(context.TODO(), ttLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		tsLayout = &tarsCrdV1Beta3.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      Resource,
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta3.TServerSpec{
				App:       App,
				Server:    Server,
				SubType:   tarsCrdV1Beta3.TARS,
				Important: 5,
				Tars: &tarsCrdV1Beta3.TServerTars{
					Template:    Template,
					Profile:     "",
					AsyncThread: 3,
					Servants: []*tarsCrdV1Beta3.TServerServant{
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
				K8S: tarsCrdV1Beta3.TServerK8S{
					AbilityAffinity: tarsCrdV1Beta3.None,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					ImagePullPolicy: k8sCoreV1.PullAlways,
					LauncherType:    tarsMeta.Background,
				},
			},
		}
		_, err = s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
	})

	ginkgo.AfterEach(func() {
		_ = s.CRDClient.CrdV1beta3().TServers(s.Namespace).Delete(context.TODO(), Resource, k8sMetaV1.DeleteOptions{})
	})

	ginkgo.It("before update", func() {
		tserver, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tserver)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:  App,
			tarsMeta.TServerNameLabel: Server,
			tarsMeta.TTemplateLabel:   Template,
			tarsMeta.TSubTypeLabel:    string(tarsCrdV1Beta3.TARS),
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
	})

	ginkgo.Context("template", func() {
		var NewTemplate = "tt.xx"
		ginkgo.BeforeEach(func() {
			ttLayout := &tarsCrdV1Beta3.TTemplate{
				ObjectMeta: k8sMetaV1.ObjectMeta{
					Name:      NewTemplate,
					Namespace: s.Namespace,
				},
				Spec: tarsCrdV1Beta3.TTemplateSpec{
					Content: "tt.xx content",
					Parent:  NewTemplate,
				},
			}
			_, err := s.CRDClient.CrdV1beta3().TTemplates(s.Namespace).Create(context.TODO(), ttLayout, k8sMetaV1.CreateOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)
		})

		ginkgo.It("not exist template", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/tars/template",
					Value: "notexit",
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/tars/template",
					Value: NewTemplate,
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			tserver, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), tserver)

			expectedLabels := map[string]string{
				tarsMeta.TServerAppLabel:  App,
				tarsMeta.TServerNameLabel: Server,
				tarsMeta.TTemplateLabel:   NewTemplate,
				tarsMeta.TSubTypeLabel:    string(tarsCrdV1Beta3.TARS),
			}
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tserver.Labels))
		})
	})

	ginkgo.It("readiness", func() {
		newReadiness := []string{"newReadiness1", "newReadiness2"}
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/k8s/readinessGates",
				Value: newReadiness,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		tserver, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tserver)
		expectedReadiness := []string{tarsMeta.TPodReadinessGate, "newReadiness1", "newReadiness2"}
		assert.Equal(ginkgo.GinkgoT(), expectedReadiness, tserver.Spec.K8S.ReadinessGates)
	})

	ginkgo.Context("release", func() {

		ginkgo.It("before release", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/replicas",
					Value: 3,
				},
			}

			bs, _ := json.Marshal(jsonPatch)
			tserver, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), int32(0), tserver.Spec.K8S.Replicas)
		})

		ginkgo.It("release", func() {

			now := k8sMetaV1.Now()
			release := tarsCrdV1Beta3.TServerRelease{
				ID:     "v1beta3",
				Image:  "www.docker.com:5050/test123:v1beta3",
				Secret: "",
				Time:   &now,
				TServerReleaseNode: &tarsCrdV1Beta3.TServerReleaseNode{
					Image:  "www.docker.com:5050/node:v1beta3",
					Secret: "tars-image-secret",
				},
			}

			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/release",
					Value: release,
				},
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/replicas",
					Value: 3,
				},
			}

			bs, _ := json.Marshal(jsonPatch)
			tserver, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), tserver.Spec.Release)
			assert.Equal(ginkgo.GinkgoT(), int32(3), tserver.Spec.K8S.Replicas)

			expectedLabels := map[string]string{
				tarsMeta.TServerAppLabel:  App,
				tarsMeta.TServerNameLabel: Server,
				tarsMeta.TServerIdLabel:   "v1beta3",
				tarsMeta.TSubTypeLabel:    string(tarsCrdV1Beta3.TARS),
			}

			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tserver.Labels))

			assert.Equal(ginkgo.GinkgoT(), map[string]string{
				"ID":     "v1beta3",
				"Image":  "www.docker.com:5050/test123:v1beta3",
				"Secret": "",
			}, map[string]string{
				"ID":     tserver.Spec.Release.ID,
				"Image":  tserver.Spec.Release.Image,
				"Secret": tserver.Spec.Release.Secret,
			})
			assert.Equal(ginkgo.GinkgoT(), *release.TServerReleaseNode, *tserver.Spec.Release.TServerReleaseNode)
		})
	})
})
