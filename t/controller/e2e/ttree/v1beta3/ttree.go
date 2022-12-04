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
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"

	"time"
)

var _ = ginkgo.Describe("test ttree", func() {

	opts := &scaffold.Options{
		Name:     "default",
		SyncTime: 800 * time.Millisecond,
	}
	s := scaffold.NewScaffold(opts)

	ginkgo.BeforeEach(func() {
		trLayout := &tarsV1Beta3.TTree{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      tarsMeta.FixedTTreeResourceName,
				Namespace: s.Namespace,
			},
			Businesses: []tarsV1Beta3.TTreeBusiness{
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
			Apps: []tarsV1Beta3.TTreeApp{
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
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(s.Namespace).Create(context.TODO(), trLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
	})

	ginkgo.It("try create ttree with unexpected name", func() {
		trLayout := &tarsV1Beta3.TTree{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "my-tree",
				Namespace: s.Namespace,
			},
			Businesses: []tarsV1Beta3.TTreeBusiness{
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
			Apps: []tarsV1Beta3.TTreeApp{
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
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(s.Namespace).Create(context.TODO(), trLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	ginkgo.It("try update business", func() {
		jsonPatch := tarsTool.JsonPatch{
			{
				OP:    tarsTool.JsonPatchReplace,
				Path:  "/businesses/0/name",
				Value: "MFrameWork",
			},
			{
				OP:    tarsTool.JsonPatchReplace,
				Path:  "/businesses/1/name",
				Value: "MBase",
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		ttree, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), ttree)
		assert.Equal(ginkgo.GinkgoT(), 2, len(ttree.Apps))
		assert.Equal(ginkgo.GinkgoT(), "", ttree.Apps[0].BusinessRef)
		assert.Equal(ginkgo.GinkgoT(), "", ttree.Apps[1].BusinessRef)
	})

	ginkgo.It("try delete business", func() {
		jsonPatch := tarsTool.JsonPatch{
			{
				OP:   tarsTool.JsonPatchRemove,
				Path: "/businesses/1",
			},
		}
		bs, _ := json.Marshal(jsonPatch)

		ttree, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), ttree)
		assert.Equal(ginkgo.GinkgoT(), 2, len(ttree.Apps))
		assert.Equal(ginkgo.GinkgoT(), "", ttree.Apps[1].BusinessRef)
	})

	ginkgo.It("try update app.businessRef", func() {
		jsonPatch := tarsTool.JsonPatch{
			{
				OP:   tarsTool.JsonPatchRemove,
				Path: "/apps/1/businessRef",
			},
		}
		bs, _ := json.Marshal(jsonPatch)

		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)

		jsonPatch = tarsTool.JsonPatch{
			{
				OP:    tarsTool.JsonPatchReplace,
				Path:  "/apps/1/businessRef",
				Value: "notExist",
			},
		}
		bs, _ = json.Marshal(jsonPatch)
		ttree, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), ttree)
		assert.Equal(ginkgo.GinkgoT(), 2, len(ttree.Apps))
		assert.Equal(ginkgo.GinkgoT(), "", ttree.Apps[1].BusinessRef)

		jsonPatch = tarsTool.JsonPatch{
			{
				OP:    tarsTool.JsonPatchReplace,
				Path:  "/apps/1/businessRef",
				Value: "Framework",
			},
		}
		bs, _ = json.Marshal(jsonPatch)
		ttree, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(s.Namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), ttree)
		assert.Equal(ginkgo.GinkgoT(), 2, len(ttree.Apps))
		assert.Equal(ginkgo.GinkgoT(), "Framework", ttree.Apps[1].BusinessRef)
	})

	ginkgo.It("try create tserver", func() {
		tsLayout := &tarsV1Beta3.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			Spec: tarsV1Beta3.TServerSpec{
				App:       "Test",
				Server:    "TestServer",
				SubType:   "tars",
				Important: 1,
				Normal: &tarsV1Beta3.TServerNormal{
					Ports: []*tarsV1Beta3.TServerPort{},
				},
				K8S: tarsV1Beta3.TServerK8S{
					DaemonSet:       false,
					AbilityAffinity: tarsV1Beta3.AppRequired,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					LauncherType:    tarsMeta.Background,
					ImagePullPolicy: k8sCoreV1.PullAlways,
				},
			},
		}
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)

		time.Sleep(time.Second * 1)
		ttree, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(s.Namespace).Get(context.TODO(), tarsMeta.FixedTTreeResourceName, k8sMetaV1.GetOptions{})
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
