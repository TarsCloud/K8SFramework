package v1beta3

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1Beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"time"
)

var _ = ginkgo.Describe("test timage", func() {
	opts := &scaffold.Options{
		Name:     "default",
		SyncTime: 800 * time.Millisecond,
	}
	s := scaffold.NewScaffold(opts)

	ginkgo.It("try create timage", func() {
		tiLayout := &tarsV1Beta3.TImage{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			ImageType:     "server",
			SupportedType: []string{"go", "cpp"},
			Releases: []*tarsV1Beta3.TImageRelease{
				{
					ID:    "202201",
					Image: "testserver:v1",
				},
			},
		}
		timage, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(s.Namespace).Create(context.TODO(), tiLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), timage)
		assert.NotNil(ginkgo.GinkgoT(), timage.Releases[0].CreateTime)
	})

	ginkgo.It("try create/update timage with duplicate id ", func() {
		tiLayout := &tarsV1Beta3.TImage{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			ImageType:     "server",
			SupportedType: []string{"go", "cpp"},
			Releases: []*tarsV1Beta3.TImageRelease{
				{
					ID:    "202201",
					Image: "testserver:v1",
				},
				{
					ID:    "202201",
					Image: "testserver:v1-2",
				},
				{
					ID:    "202202",
					Image: "testserver:v2",
				},
				{
					ID:    "202202",
					Image: "testserver:v2-2",
				},
				{
					ID:    "202203",
					Image: "testserver:v3",
				},
				{
					ID:    "202202",
					Image: "testserver:v3-2",
				},
			},
		}
		timage, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(s.Namespace).Create(context.TODO(), tiLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), timage)
		assert.Equal(ginkgo.GinkgoT(), 3, len(timage.Releases))
		assert.Equal(ginkgo.GinkgoT(), "testserver:v1", timage.Releases[0].Image)
		assert.Equal(ginkgo.GinkgoT(), "testserver:v2", timage.Releases[1].Image)
		assert.Equal(ginkgo.GinkgoT(), "testserver:v3", timage.Releases[2].Image)

		jsonPatch := tarsTool.JsonPatch{
			{
				OP:    tarsTool.JsonPatchReplace,
				Path:  "/releases/1/id",
				Value: "202201",
			},
			{
				OP:    tarsTool.JsonPatchReplace,
				Path:  "/releases/2/id",
				Value: "202201",
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		timage, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(s.Namespace).Patch(context.TODO(), "test-testserver", patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), timage)
		assert.Equal(ginkgo.GinkgoT(), 1, len(timage.Releases))
		assert.Equal(ginkgo.GinkgoT(), "testserver:v1", timage.Releases[0].Image)
	})
})
