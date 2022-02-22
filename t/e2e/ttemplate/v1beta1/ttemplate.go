package v1beta1

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	crdV1Beta2 "k8s.tars.io/api/crd/v1beta1"
	crdMeta "k8s.tars.io/api/meta"
	"time"
)

var _ = ginkgo.Describe("test app level config", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
	}
	s := scaffold.NewScaffold(opts)

	defaultTTLayout := &crdV1Beta2.TTemplate{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      "tt.default",
			Namespace: s.Namespace,
		},
		Spec: crdV1Beta2.TTemplateSpec{
			Content: "tt.default content",
			Parent:  "tt.default",
		},
	}

	cppTTLayout := &crdV1Beta2.TTemplate{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      "tt.cpp",
			Namespace: s.Namespace,
		},
		Spec: crdV1Beta2.TTemplateSpec{
			Content: "tt.cpp content",
			Parent:  "tt.default",
		},
	}

	ginkgo.BeforeEach(func() {
		defaultTT, err := s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Create(context.TODO(), defaultTTLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), defaultTT)

		cppTT, err := s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Create(context.TODO(), cppTTLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), cppTT)
	})

	ginkgo.It("check labels", func() {
		defaultTT, err := s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Get(context.TODO(), "tt.default", k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), defaultTT)

		if defaultTT.Labels != nil {
			_, ok := defaultTT.Labels[crdMeta.ParentLabel]
			assert.False(ginkgo.GinkgoT(), ok)
		}

		cppTT, err := s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Get(context.TODO(), "tt.cpp", k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), defaultTT)
		exceptedCppTTLabels := map[string]string{
			crdMeta.ParentLabel: "tt.default",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedCppTTLabels, cppTT.Labels))
	})

	ginkgo.It("try create ttemplate and parent not exist", func() {
		xxLayout := &crdV1Beta2.TTemplate{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "tt.xxx",
				Namespace: s.Namespace,
			},
			Spec: crdV1Beta2.TTemplateSpec{
				Content: "tt.xxx Content",
				Parent:  "tt.parent",
			},
		}
		_, err := s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Create(context.TODO(), xxLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	ginkgo.It("try update ttemplate content", func() {
		jsonPatch := crdMeta.JsonPatch{
			{
				OP:    crdMeta.JsonPatchReplace,
				Path:  "/spec/content",
				Value: "new content",
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		defaultXX, err := s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Patch(context.TODO(), "tt.default", patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), defaultXX)
		assert.True(ginkgo.GinkgoT(), defaultXX.Spec.Content == "new content")

		cppTT, err := s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Patch(context.TODO(), "tt.cpp", patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), cppTT)
		assert.True(ginkgo.GinkgoT(), cppTT.Spec.Content == "new content")
	})

	ginkgo.It("try delete ttemplate ", func() {
		err := s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Delete(context.TODO(), "tt.default", k8sMetaV1.DeleteOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)

		tsLayout := &crdV1Beta2.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			Spec: crdV1Beta2.TServerSpec{
				App:       "Test",
				Server:    "TestServer",
				SubType:   "tars",
				Important: 1,
				Tars: &crdV1Beta2.TServerTars{
					Template:    "tt.cpp",
					Profile:     "",
					AsyncThread: 3,
					Servants: []*crdV1Beta2.TServerServant{
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
				K8S: crdV1Beta2.TServerK8S{
					DaemonSet:       false,
					AbilityAffinity: crdV1Beta2.AppRequired,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
				},
				Release: &crdV1Beta2.TServerRelease{
					ID:    "202201",
					Image: "xxxx.com/test.teserver:v1",
				},
			},
		}
		_, err = s.CRDClient.CrdV1beta1().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)

		err = s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Delete(context.TODO(), "tt.cpp", k8sMetaV1.DeleteOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)

		err = s.CRDClient.CrdV1beta1().TServers(s.Namespace).Delete(context.TODO(), "test-testserver", k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(time.Second)

		err = s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Delete(context.TODO(), "tt.cpp", k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)

		err = s.CRDClient.CrdV1beta1().TTemplates(s.Namespace).Delete(context.TODO(), "tt.default", k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
	})
})
