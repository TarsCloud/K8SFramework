package v1beta3

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"

	"strings"
	"time"
)

var _ = ginkgo.Describe("test server level config", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  1500 * time.Millisecond,
	}
	s := scaffold.NewScaffold(opts)

	ResourceName := "server.config.1"
	ServerApp := "Test"
	ServerName := "TestServer"
	ConfigName := "server.conf"
	ConfigContent := "Config Content"

	ginkgo.BeforeEach(func() {
		tconfigLayout := &tarsCrdV1beta3.TConfig{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      ResourceName,
				Namespace: s.Namespace,
			},
			App:           ServerApp,
			Server:        ServerName,
			PodSeq:        "m",
			ConfigName:    ConfigName,
			ConfigContent: ConfigContent,
			Activated:     false,
			UpdateTime:    k8sMetaV1.Now(),
		}
		tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Create(context.TODO(), tconfigLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)
	})

	ginkgo.It("valid labels ", func() {
		tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:       ServerApp,
			tarsMeta.TServerNameLabel:      ServerName,
			tarsMeta.TConfigPodSeqLabel:    "m",
			tarsMeta.TConfigActivatedLabel: "false",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
	})

	ginkgo.It("remove labels ", func() {
		tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:       ServerApp,
			tarsMeta.TServerNameLabel:      ServerName,
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
			tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
		}
	})

	ginkgo.It("update labels ", func() {
		tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:       ServerApp,
			tarsMeta.TServerNameLabel:      ServerName,
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
			tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
		}
	})

	ginkgo.It("update immutable filed", func() {
		tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
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
			_, err = s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
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
		tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)
		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:       ServerApp,
			tarsMeta.TServerNameLabel:      ServerName,
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
		tconfig, err = s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	ginkgo.Context("new version of server level config", func() {
		NewResourceName := "app.config.2"
		NewConfigContent := "New Config Content"
		newTConfigLayout := &tarsCrdV1beta3.TConfig{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      NewResourceName,
				Namespace: s.Namespace,
			},
			App:           ServerApp,
			Server:        ServerName,
			PodSeq:        "m",
			ConfigName:    ConfigName,
			ConfigContent: NewConfigContent,
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
			tconfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), tconfig)

			exceptedBeforeCreateNewLabels := map[string]string{
				tarsMeta.TServerAppLabel:       ServerApp,
				tarsMeta.TServerNameLabel:      ServerName,
				tarsMeta.TConfigPodSeqLabel:    "m",
				tarsMeta.TConfigActivatedLabel: "true",
			}
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedBeforeCreateNewLabels, tconfig.Labels))
		})

		ginkgo.It("create/delete new version tconfig", func() {
			newTConfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Create(context.TODO(), newTConfigLayout, k8sMetaV1.CreateOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), newTConfig)
			exceptedNewTConfigLabels := map[string]string{
				tarsMeta.TServerAppLabel:       ServerApp,
				tarsMeta.TServerNameLabel:      ServerName,
				tarsMeta.TConfigPodSeqLabel:    "m",
				tarsMeta.TConfigActivatedLabel: "true",
			}
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedNewTConfigLabels, newTConfig.Labels))

			time.Sleep(s.Opts.SyncTime)
			exceptedAfterCreateNewLabels := map[string]string{
				tarsMeta.TServerAppLabel:       ServerApp,
				tarsMeta.TServerNameLabel:      ServerName,
				tarsMeta.TConfigPodSeqLabel:    "m",
				tarsMeta.TConfigActivatedLabel: "false",
			}
			oldTConfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), oldTConfig)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedAfterCreateNewLabels, oldTConfig.Labels))

			err = s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Delete(context.TODO(), NewResourceName, k8sMetaV1.DeleteOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)

			time.Sleep(s.Opts.SyncTime)
			oldTConfig, err = s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
			if err == nil {
				assert.True(ginkgo.GinkgoT(), k8sMetaV1.HasLabel(oldTConfig.ObjectMeta, tarsMeta.TConfigDeletingLabel))
			}
		})
	})

	ginkgo.Context("slave tconfig", func() {

		slaveResourceName := "slave.app.config"
		slaveConfigContent := "Slave Config Content"
		slaveTConfigLayout := &tarsCrdV1beta3.TConfig{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      slaveResourceName,
				Namespace: s.Namespace,
			},
			App:           ServerApp,
			Server:        ServerName,
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
			oldTConfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), oldTConfig)

			exceptedBeforeCreateNewLabels := map[string]string{
				tarsMeta.TServerAppLabel:       ServerApp,
				tarsMeta.TServerNameLabel:      ServerName,
				tarsMeta.TConfigPodSeqLabel:    "m",
				tarsMeta.TConfigActivatedLabel: "true",
			}
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedBeforeCreateNewLabels, oldTConfig.Labels))
			time.Sleep(s.Opts.SyncTime)
		})

		ginkgo.It("no master config", func() {
			slaveTConfigLayout.ConfigName = "NoMasterTConfig"
			var err error
			_, err = s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Create(context.TODO(), slaveTConfigLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		})

		ginkgo.It("create/delete slave tconfig", func() {
			slaveTConfigLayout.ConfigName = ConfigName
			slaveTConfig, err := s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Create(context.TODO(), slaveTConfigLayout, k8sMetaV1.CreateOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), slaveTConfig)
			slaveConfigExceptedLabels := map[string]string{
				tarsMeta.TServerAppLabel:       ServerApp,
				tarsMeta.TServerNameLabel:      ServerName,
				tarsMeta.TConfigPodSeqLabel:    "1",
				tarsMeta.TConfigActivatedLabel: "true",
			}
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(slaveConfigExceptedLabels, slaveTConfig.Labels))

			err = s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Delete(context.TODO(), ResourceName, k8sMetaV1.DeleteOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)

			err = s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Delete(context.TODO(), slaveResourceName, k8sMetaV1.DeleteOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			err = s.CRDClient.CrdV1beta3().TConfigs(s.Namespace).Delete(context.TODO(), ResourceName, k8sMetaV1.DeleteOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
		})
	})
})
