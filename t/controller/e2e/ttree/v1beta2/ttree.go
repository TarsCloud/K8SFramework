package v1beta2

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1Beta2 "k8s.tars.io/crd/v1beta2"
	tarsMeta "k8s.tars.io/meta"

	"time"
)

var _ = ginkgo.Describe("test ttree", func() {

	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  1500 * time.Millisecond,
	}
	s := scaffold.NewScaffold(opts)

	ginkgo.BeforeEach(func() {
		trLayout := &tarsCrdV1Beta2.TTree{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      tarsMeta.FixedTTreeResourceName,
				Namespace: s.Namespace,
			},
			Businesses: []tarsCrdV1Beta2.TTreeBusiness{
				{
					Name:       "Framework",
					Show:       "框架服务",
					Weight:     3,
					CreateTime: k8sMetaV1.Now(),
				},
				{
					Name:       "Base",
					Show:       "基础服务",
					Weight:     5,
					CreateTime: k8sMetaV1.Now(),
				},
			},
			Apps: []tarsCrdV1Beta2.TTreeApp{
				{
					Name:        "test1",
					BusinessRef: "Framework",
					CreateTime:  k8sMetaV1.Now(),
				},
				{
					Name:        "test2",
					BusinessRef: "Base",
					CreateTime:  k8sMetaV1.Now(),
				},
			},
		}
		_, err := s.CRDClient.CrdV1beta2().TTrees(s.Namespace).Create(context.TODO(), trLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
	})

	ginkgo.It("try create ttree with unexpected name", func() {
		trLayout := &tarsCrdV1Beta2.TTree{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "my-tree",
				Namespace: s.Namespace,
			},
			Businesses: []tarsCrdV1Beta2.TTreeBusiness{
				{
					Name:       "NewFramework",
					Show:       "框架服务",
					Weight:     3,
					CreateTime: k8sMetaV1.Now(),
				},
				{
					Name:       "NewBase",
					Show:       "基础服务",
					Weight:     5,
					CreateTime: k8sMetaV1.Now(),
				},
			},
			Apps: []tarsCrdV1Beta2.TTreeApp{
				{
					Name:        "NewTest1",
					BusinessRef: "NewFramework",
					CreateTime:  k8sMetaV1.Now(),
				},
				{
					Name:        "NewTest2",
					BusinessRef: "NewBase",
					CreateTime:  k8sMetaV1.Now(),
				},
			},
		}
		_, err := s.CRDClient.CrdV1beta2().TTrees(s.Namespace).Create(context.TODO(), trLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	ginkgo.It("try update business", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/businesses/0/name",
				Value: "MFrameWork",
			},
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/businesses/1/name",
				Value: "MBase",
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		ttree, err := s.CRDClient.CrdV1beta2().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), ttree)
		assert.Equal(ginkgo.GinkgoT(), 2, len(ttree.Apps))
		assert.Equal(ginkgo.GinkgoT(), "", ttree.Apps[0].BusinessRef)
		assert.Equal(ginkgo.GinkgoT(), "", ttree.Apps[1].BusinessRef)
	})

	ginkgo.It("try delete business", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchRemove,
				Path: "/businesses/1",
			},
		}
		bs, _ := json.Marshal(jsonPatch)

		ttree, err := s.CRDClient.CrdV1beta2().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), ttree)
		assert.Equal(ginkgo.GinkgoT(), 2, len(ttree.Apps))
		assert.Equal(ginkgo.GinkgoT(), "", ttree.Apps[1].BusinessRef)
	})

	ginkgo.It("try update app.businessRef", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchRemove,
				Path: "/apps/1/businessRef",
			},
		}
		bs, _ := json.Marshal(jsonPatch)

		_, err := s.CRDClient.CrdV1beta2().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)

		jsonPatch = tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/apps/1/businessRef",
				Value: "notExist",
			},
		}
		bs, _ = json.Marshal(jsonPatch)
		ttree, err := s.CRDClient.CrdV1beta2().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), ttree)
		assert.Equal(ginkgo.GinkgoT(), 2, len(ttree.Apps))
		assert.Equal(ginkgo.GinkgoT(), "", ttree.Apps[1].BusinessRef)

		jsonPatch = tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/apps/1/businessRef",
				Value: "Framework",
			},
		}
		bs, _ = json.Marshal(jsonPatch)
		ttree, err = s.CRDClient.CrdV1beta2().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), ttree)
		assert.Equal(ginkgo.GinkgoT(), 2, len(ttree.Apps))
		assert.Equal(ginkgo.GinkgoT(), "Framework", ttree.Apps[1].BusinessRef)
	})

	ginkgo.It("try create tserver", func() {
		tsLayout := &tarsCrdV1Beta2.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta2.TServerSpec{
				App:       "Test",
				Server:    "TestServer",
				SubType:   "tars",
				Important: 1,
				Normal: &tarsCrdV1Beta2.TServerNormal{
					Ports: []*tarsCrdV1Beta2.TServerPort{},
				},
				K8S: tarsCrdV1Beta2.TServerK8S{
					DaemonSet:       false,
					AbilityAffinity: tarsCrdV1Beta2.AppRequired,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					LauncherType:    tarsMeta.Background,
					ImagePullPolicy: k8sCoreV1.PullAlways,
				},
			},
		}
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)

		time.Sleep(time.Second * 1)
		ttree, err := s.CRDClient.CrdV1beta2().TTrees(s.Namespace).Get(context.TODO(), tarsMeta.FixedTTreeResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Equal(ginkgo.GinkgoT(), 3, len(ttree.Apps))
		assert.True(ginkgo.GinkgoT(), func() bool {
			for _, v := range ttree.Apps {
				if v.Name == "Test" && v.BusinessRef == "" {
					return true
				}
			}
			return false
		}())
	})
})
