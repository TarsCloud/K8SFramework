package v1beta1

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1Beta1 "k8s.tars.io/crd/v1beta1"
	tarsMetaTools "k8s.tars.io/meta/tools"
	"time"
)

var _ = ginkgo.Describe("test timage", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  1500 * time.Millisecond,
	}
	s := scaffold.NewScaffold(opts)

	ginkgo.It("try create timage", func() {
		tiLayout := &tarsCrdV1Beta1.TImage{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			ImageType:     "server",
			SupportedType: []string{"go", "cpp"},
			Releases: []*tarsCrdV1Beta1.TImageRelease{
				{
					ID:    "202201",
					Image: "testserver:v1",
				},
			},
		}
		timage, err := s.CRDClient.CrdV1beta1().TImages(s.Namespace).Create(context.TODO(), tiLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), timage)
		assert.NotNil(ginkgo.GinkgoT(), timage.Releases[0].CreateTime)
	})

	ginkgo.It("try create/update timage with duplicate id ", func() {
		tiLayout := &tarsCrdV1Beta1.TImage{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			ImageType:     "server",
			SupportedType: []string{"go", "cpp"},
			Releases: []*tarsCrdV1Beta1.TImageRelease{
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
		timage, err := s.CRDClient.CrdV1beta1().TImages(s.Namespace).Create(context.TODO(), tiLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), timage)
		assert.Equal(ginkgo.GinkgoT(), 3, len(timage.Releases))
		assert.Equal(ginkgo.GinkgoT(), "testserver:v1", timage.Releases[0].Image)
		assert.Equal(ginkgo.GinkgoT(), "testserver:v2", timage.Releases[1].Image)
		assert.Equal(ginkgo.GinkgoT(), "testserver:v3", timage.Releases[2].Image)

		jsonPatch := tarsMetaTools.JsonPatch{
			{
				OP:    tarsMetaTools.JsonPatchReplace,
				Path:  "/releases/1/id",
				Value: "202201",
			},
			{
				OP:    tarsMetaTools.JsonPatchReplace,
				Path:  "/releases/2/id",
				Value: "202201",
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		timage, err = s.CRDClient.CrdV1beta1().TImages(s.Namespace).Patch(context.TODO(), "test-testserver", patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), timage)
		assert.Equal(ginkgo.GinkgoT(), 1, len(timage.Releases))
		assert.Equal(ginkgo.GinkgoT(), "testserver:v1", timage.Releases[0].Image)
	})
})
