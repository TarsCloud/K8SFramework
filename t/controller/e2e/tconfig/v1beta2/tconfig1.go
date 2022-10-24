package v1beta2

import (
	"context"
	"e2e/scaffold"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1beta2 "k8s.tars.io/crd/v1beta2"
	tarsMeta "k8s.tars.io/meta"

	"strings"
	"time"
)

var _ = ginkgo.Describe("test app level config", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  800 * time.Millisecond,
	}
	s := scaffold.NewScaffold(opts)

	ResourceName := "app.config.1"
	ServerApp := "Test"
	ConfigName := "app.conf"
	ConfigContent := "Config Content"

	ginkgo.BeforeEach(func() {
		appConfig := &tarsCrdV1beta2.TConfig{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      ResourceName,
				Namespace: s.Namespace,
			},
			App:           ServerApp,
			PodSeq:        "m",
			ConfigName:    ConfigName,
			ConfigContent: ConfigContent,
			Activated:     false,
			UpdateTime:    k8sMetaV1.Now(),
		}
		_, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Create(context.TODO(), appConfig, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)
	})

	ginkgo.It("valid labels ", func() {
		tconfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)
		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:       ServerApp,
			tarsMeta.TServerNameLabel:      "",
			tarsMeta.TConfigPodSeqLabel:    "m",
			tarsMeta.TConfigActivatedLabel: "false",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
	})

	ginkgo.It("remove labels ", func() {
		tconfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:       ServerApp,
			tarsMeta.TServerNameLabel:      "",
			tarsMeta.TConfigPodSeqLabel:    "m",
			tarsMeta.TConfigActivatedLabel: "false",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))

		tryRemoveLabels := []string{tarsMeta.TServerAppLabel, tarsMeta.TServerNameLabel, tarsMeta.TConfigPodSeqLabel, tarsMeta.TConfigVersionLabel}
		for _, v := range tryRemoveLabels {
			jsonPath := tarsMeta.JsonPatch{
				{
					OP:   tarsMeta.JsonPatchRemove,
					Path: "/metadata/labels/" + strings.Replace(v, "/", "~1", 1),
				},
			}
			bs, _ := json.Marshal(jsonPath)
			tconfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
		}
	})

	ginkgo.It("update labels ", func() {
		tconfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:       ServerApp,
			tarsMeta.TServerNameLabel:      "",
			tarsMeta.TConfigPodSeqLabel:    "m",
			tarsMeta.TConfigActivatedLabel: "false",
		}

		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))

		tryUpdateLabels := []string{tarsMeta.TServerAppLabel, tarsMeta.TServerNameLabel, tarsMeta.TConfigPodSeqLabel, tarsMeta.TConfigVersionLabel}
		for _, v := range tryUpdateLabels {
			jsonPath := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/metadata/labels/" + strings.Replace(v, "/", "~1", 1),
					Value: scaffold.RandStringRunes(5),
				},
			}
			bs, _ := json.Marshal(jsonPath)
			tconfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
		}
	})

	ginkgo.It("update immutable filed", func() {
		tconfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		immutableFields := map[string]string{
			"/app":           "NewApp",
			"/server":        "NewServer",
			"/podSeq":        "1",
			"/configName":    "NewConfigName",
			"/configContent": "NewContent",
		}
		for k, v := range immutableFields {
			jsonPath := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  k,
					Value: v,
				},
			}
			bs, _ := json.Marshal(jsonPath)
			_, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		}
	})

	ginkgo.It("activated/inactivated tconfig", func() {
		jsonPath := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: true,
			},
		}

		bs, _ := json.Marshal(jsonPath)
		tconfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)
		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:       ServerApp,
			tarsMeta.TServerNameLabel:      "",
			tarsMeta.TConfigPodSeqLabel:    "m",
			tarsMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))

		jsonPath = tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: false,
			},
		}
		bs, _ = json.Marshal(jsonPath)
		tconfig, err = s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	ginkgo.Context("new version", func() {
		ginkgo.BeforeEach(func() {
			jsonPath := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/activated",
					Value: true,
				},
			}
			bs, _ := json.Marshal(jsonPath)
			oldTConfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), oldTConfig)

			exceptedBeforeCreateNewLabels := map[string]string{
				tarsMeta.TServerAppLabel:       ServerApp,
				tarsMeta.TServerNameLabel:      "",
				tarsMeta.TConfigPodSeqLabel:    "m",
				tarsMeta.TConfigActivatedLabel: "true",
			}
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedBeforeCreateNewLabels, oldTConfig.Labels))
		})

		NewResourceName := "app.config.2"
		NewConfigContent := "New Config Content"
		newTConfigLayout := &tarsCrdV1beta2.TConfig{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      NewResourceName,
				Namespace: s.Namespace,
			},
			App:           ServerApp,
			PodSeq:        "m",
			ConfigName:    ConfigName,
			ConfigContent: NewConfigContent,
			Activated:     true,
			UpdateTime:    k8sMetaV1.Now(),
		}

		ginkgo.It("create/delete new version", func() {
			newTConfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Create(context.TODO(), newTConfigLayout, k8sMetaV1.CreateOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), newTConfig)
			exceptedNewTConfigLabels := map[string]string{
				tarsMeta.TServerAppLabel:       ServerApp,
				tarsMeta.TServerNameLabel:      "",
				tarsMeta.TConfigPodSeqLabel:    "m",
				tarsMeta.TConfigActivatedLabel: "true",
			}
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedNewTConfigLabels, newTConfig.Labels))

			time.Sleep(s.Opts.SyncTime)
			oldTConfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), oldTConfig)
			expected := !oldTConfig.Activated || k8sMetaV1.HasLabel(oldTConfig.ObjectMeta, tarsMeta.TConfigDeactivateLabel)
			if !expected {
				bs, _ := json.Marshal(oldTConfig)
				fmt.Printf("get unexpected tconfig: %s\n", string(bs))
			}
			assert.True(ginkgo.GinkgoT(), expected)

			err = s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Delete(context.TODO(), NewResourceName, k8sMetaV1.DeleteOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "during deletion guard time"))

			time.Sleep(50 * time.Second) //skip tconfig deletion guard time
			err = s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Delete(context.TODO(), NewResourceName, k8sMetaV1.DeleteOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)

			time.Sleep(s.Opts.SyncTime)
			oldTConfig, err = s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
			if err == nil {
				assert.True(ginkgo.GinkgoT(), k8sMetaV1.HasLabel(oldTConfig.ObjectMeta, tarsMeta.TConfigDeletingLabel))
			}
		})
	})

	ginkgo.Context("new slave tconfig", func() {
		slaveResourceName := "slave.app.config"
		slaveConfigContent := "Slave Config Content"
		slaveTConfigLayout := &tarsCrdV1beta2.TConfig{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      slaveResourceName,
				Namespace: s.Namespace,
			},
			App:           ServerApp,
			PodSeq:        "1",
			ConfigName:    ConfigName,
			ConfigContent: slaveConfigContent,
			Activated:     true,
			UpdateTime:    k8sMetaV1.Now(),
		}

		ginkgo.BeforeEach(func() {
			jsonPath := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/activated",
					Value: true,
				},
			}
			bs, _ := json.Marshal(jsonPath)
			tconfig, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), tconfig)

			exceptedBeforeCreateNewLabels := map[string]string{
				tarsMeta.TServerAppLabel:       ServerApp,
				tarsMeta.TServerNameLabel:      "",
				tarsMeta.TConfigPodSeqLabel:    "m",
				tarsMeta.TConfigActivatedLabel: "true",
			}
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedBeforeCreateNewLabels, tconfig.Labels))
			time.Sleep(s.Opts.SyncTime)
		})

		ginkgo.It("create slave tconfig", func() {
			slaveTConfigLayout.ConfigName = ConfigName
			_, err := s.CRDClient.CrdV1beta2().TConfigs(s.Namespace).Create(context.TODO(), slaveTConfigLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		})
	})
})
