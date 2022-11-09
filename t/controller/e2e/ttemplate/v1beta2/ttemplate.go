package v1beta2

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsTarsV1beta2 "k8s.tars.io/apis/tars/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"time"
)

var _ = ginkgo.Describe("ttemplate", func() {
	opts := &scaffold.Options{
		Name:     "default",
		SyncTime: 800 * time.Millisecond,
	}
	s := scaffold.NewScaffold(opts)

	defaultTTLayout := &tarsTarsV1beta2.TTemplate{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      "tt.default",
			Namespace: s.Namespace,
		},
		Spec: tarsTarsV1beta2.TTemplateSpec{
			Content: "tt.default content",
			Parent:  "tt.default",
		},
	}

	cppTTLayout := &tarsTarsV1beta2.TTemplate{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      "tt.cpp",
			Namespace: s.Namespace,
		},
		Spec: tarsTarsV1beta2.TTemplateSpec{
			Content: "tt.cpp content",
			Parent:  "tt.default",
		},
	}

	ginkgo.BeforeEach(func() {
		defaultTT, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Create(context.TODO(), defaultTTLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), defaultTT)
		time.Sleep(s.Opts.SyncTime)
		cppTT, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Create(context.TODO(), cppTTLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), cppTT)
		time.Sleep(s.Opts.SyncTime)
	})

	ginkgo.It("check labels", func() {
		defaultTT, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Get(context.TODO(), defaultTTLayout.Name, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), defaultTT)

		if defaultTT.Labels != nil {
			_, ok := defaultTT.Labels[tarsMeta.TTemplateParentLabel]
			assert.False(ginkgo.GinkgoT(), ok)
		}

		cppTT, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Get(context.TODO(), cppTTLayout.Name, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), defaultTT)
		exceptedCppTTLabels := map[string]string{
			tarsMeta.TTemplateParentLabel: "tt.default",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedCppTTLabels, cppTT.Labels))
	})

	ginkgo.It("create ttemplate and parent not exist", func() {
		xxLayout := &tarsTarsV1beta2.TTemplate{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "tt.xxx",
				Namespace: s.Namespace,
			},
			Spec: tarsTarsV1beta2.TTemplateSpec{
				Content: "tt.xxx Content",
				Parent:  "tt.parent",
			},
		}
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Create(context.TODO(), xxLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	ginkgo.It("update ttemplate content", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/content",
				Value: "new content",
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		defaultXX, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Patch(context.TODO(), defaultTTLayout.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), defaultXX)
		assert.True(ginkgo.GinkgoT(), defaultXX.Spec.Content == "new content")

		cppTT, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Patch(context.TODO(), cppTTLayout.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), cppTT)
		assert.True(ginkgo.GinkgoT(), cppTT.Spec.Content == "new content")
	})

	ginkgo.Context("delete ttemplate ", func() {
		ginkgo.It("reference by another ttemplate ", func() {
			err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Delete(context.TODO(), defaultTTLayout.Name, k8sMetaV1.DeleteOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		})

		ginkgo.It("reference by another tserver", func() {
			tsLayout := &tarsTarsV1beta2.TServer{
				ObjectMeta: k8sMetaV1.ObjectMeta{
					Name:      "test-testserver",
					Namespace: s.Namespace,
				},
				Spec: tarsTarsV1beta2.TServerSpec{
					App:       "Test",
					Server:    "TestServer",
					SubType:   "tars",
					Important: 1,
					Tars: &tarsTarsV1beta2.TServerTars{
						Template:    cppTTLayout.Name,
						Profile:     "",
						AsyncThread: 3,
						Servants: []*tarsTarsV1beta2.TServerServant{
							{
								Name:       "TestObj",
								Port:       10000,
								Thread:     3,
								Connection: 3000,
								Capacity:   3000,
								Timeout:    3000,
								IsTars:     true,
								IsTcp:      true,
							},
						},
					},
					K8S: tarsTarsV1beta2.TServerK8S{
						NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
						AbilityAffinity: tarsTarsV1beta2.AppRequired,
						Resources:       k8sCoreV1.ResourceRequirements{},
						UpdateStrategy:  k8sAppsV1.StatefulSetUpdateStrategy{},
						ImagePullPolicy: k8sCoreV1.PullAlways,
						LauncherType:    tarsMeta.Background,
					},
					Release: &tarsTarsV1beta2.TServerRelease{
						ID:    "202201",
						Image: "xxxx.com/test.teserver:v1",
						TServerReleaseNode: &tarsTarsV1beta2.TServerReleaseNode{
							Image: "xxxx.com/tafnode:v1",
						},
					},
				},
			}
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			err = tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Delete(context.TODO(), cppTTLayout.Name, k8sMetaV1.DeleteOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		})

		ginkgo.It("not referenced", func() {
			err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Delete(context.TODO(), cppTTLayout.Name, k8sMetaV1.DeleteOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			err = tarsRuntime.Clients.CrdClient.TarsV1beta2().TTemplates(s.Namespace).Delete(context.TODO(), defaultTTLayout.Name, k8sMetaV1.DeleteOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
		})
	})
})
