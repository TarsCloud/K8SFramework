package v1beta1

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	crdV1Beta2 "k8s.tars.io/api/crd/v1beta1"
	crdMeta "k8s.tars.io/api/meta"
	"strings"
	"time"
)

var _ = ginkgo.Describe("test app level config", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
	}
	s := scaffold.NewScaffold(opts)

	ResourceName := "app.config.1"
	ServerApp := "Test"
	ConfigName := "app.conf"
	ConfigContent := "Config Content"

	ginkgo.BeforeEach(func() {
		appConfig := &crdV1Beta2.TConfig{
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
		_, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), appConfig, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
	})

	ginkgo.It("check labels ", func() {
		tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		expectedLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      "",
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "false",
		}

		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))

		willRemoveLabels := []string{crdMeta.TServerAppLabel, crdMeta.TServerNameLabel, crdMeta.TConfigPodSeqLabel, crdMeta.TConfigVersionLabel}
		for _, v := range willRemoveLabels {
			jsonPath := crdMeta.JsonPatch{
				{
					OP:   crdMeta.JsonPatchRemove,
					Path: "/metadata/labels/" + strings.Replace(v, "/", "~1", 1),
				},
			}
			bs, _ := json.Marshal(jsonPath)
			tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
		}

		willUpdateLabels := []string{crdMeta.TServerAppLabel, crdMeta.TServerNameLabel, crdMeta.TConfigPodSeqLabel, crdMeta.TConfigVersionLabel}
		for _, v := range willUpdateLabels {
			jsonPath := crdMeta.JsonPatch{
				{
					OP:    crdMeta.JsonPatchReplace,
					Path:  "/metadata/labels/" + strings.Replace(v, "/", "~1", 1),
					Value: "xxx",
				},
			}
			bs, _ := json.Marshal(jsonPath)
			tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
		}
	})

	ginkgo.It("try update immutable filed", func() {
		tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		immutableFileds := map[string]string{
			"/app":           "NewApp",
			"/server":        "NewServer",
			"/podSeq":        "1",
			"/configName":    "NewConfigName",
			"/configContent": "NewContent",
		}
		for k, v := range immutableFileds {
			jsonPath := crdMeta.JsonPatch{
				{
					OP:    crdMeta.JsonPatchReplace,
					Path:  k,
					Value: v,
				},
			}
			bs, _ := json.Marshal(jsonPath)
			_, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		}
	})

	ginkgo.It("try activated/inactivated config", func() {
		jsonPath := crdMeta.JsonPatch{
			{
				OP:    crdMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: true,
			},
		}

		bs, _ := json.Marshal(jsonPath)
		tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)
		expectedLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      "",
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))

		jsonPath = crdMeta.JsonPatch{
			{
				OP:    crdMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: false,
			},
		}
		bs, _ = json.Marshal(jsonPath)
		tconfig, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	NewResourceName := "app.config.2"
	NewConfigContent := "New Config Content"
	ginkgo.It("try create&delete new version app level config", func() {
		jsonPath := crdMeta.JsonPatch{
			{
				OP:    crdMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPath)
		oldTConfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), oldTConfig)

		exceptedBeforeCreateNewLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      "",
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedBeforeCreateNewLabels, oldTConfig.Labels))

		newTConfigLayout := &crdV1Beta2.TConfig{
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
		newTconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), newTConfigLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), newTconfig)

		exceptedNewTconfigLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      "",
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedNewTconfigLabels, newTconfig.Labels))

		exceptedAfterCreateNewLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      "",
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "false",
		}
		time.Sleep(time.Second * 1)
		oldTConfig, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), oldTConfig)
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedAfterCreateNewLabels, oldTConfig.Labels))

		err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Delete(context.TODO(), NewResourceName, k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)

		time.Sleep(time.Second * 1)
		oldTConfig, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})
})

var _ = ginkgo.Describe("test server level config", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
	}
	s := scaffold.NewScaffold(opts)

	ResourceName := "server.config.1"
	ServerApp := "Test"
	ServerName := "TestServer"
	ConfigName := "server.conf"
	ConfigContent := "Config Content"

	ginkgo.BeforeEach(func() {
		tconfigLayout := &crdV1Beta2.TConfig{
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
		tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), tconfigLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)
	})

	ginkgo.It("check labels ", func() {
		tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		expectedLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      ServerName,
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "false",
		}

		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))

		willRemoveLabels := []string{crdMeta.TServerAppLabel, crdMeta.TServerNameLabel, crdMeta.TConfigPodSeqLabel, crdMeta.TConfigVersionLabel}
		for _, v := range willRemoveLabels {
			jsonPath := crdMeta.JsonPatch{
				{
					OP:   crdMeta.JsonPatchRemove,
					Path: "/metadata/labels/" + strings.Replace(v, "/", "~1", 1),
				},
			}
			bs, _ := json.Marshal(jsonPath)
			tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), tconfig)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
		}

		willUpdateLabels := []string{crdMeta.TServerAppLabel, crdMeta.TServerNameLabel, crdMeta.TConfigPodSeqLabel, crdMeta.TConfigVersionLabel}
		for _, v := range willUpdateLabels {
			jsonPath := crdMeta.JsonPatch{
				{
					OP:    crdMeta.JsonPatchReplace,
					Path:  "/metadata/labels/" + strings.Replace(v, "/", "~1", 1),
					Value: "xxx",
				},
			}
			bs, _ := json.Marshal(jsonPath)
			tconfig, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), tconfig)
			assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))
		}
	})

	ginkgo.It("try update immutable filed", func() {
		tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		immutableFileds := map[string]string{
			"/app":           "NewApp",
			"/server":        "NewServer",
			"/podSeq":        "1",
			"/configName":    "NewConfigName",
			"/configContent": "NewContent",
		}
		for k, v := range immutableFileds {
			jsonPath := crdMeta.JsonPatch{
				{
					OP:    crdMeta.JsonPatchReplace,
					Path:  k,
					Value: v,
				},
			}
			bs, _ := json.Marshal(jsonPath)
			_, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
		}
	})

	ginkgo.It("try activated/inactivated server level config", func() {
		jsonPath := crdMeta.JsonPatch{
			{
				OP:    crdMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPath)
		tconfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tconfig)

		expectedLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      ServerName,
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, tconfig.Labels))

		jsonPath = crdMeta.JsonPatch{
			{
				OP:    crdMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: false,
			},
		}
		bs, _ = json.Marshal(jsonPath)
		tconfig, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), tconfig.Name, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	NewResourceName := "server.config.2"
	NewConfigContent := "New Config Content"

	ginkgo.It("try create&delete new version server config", func() {
		jsonPath := crdMeta.JsonPatch{
			{
				OP:    crdMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPath)
		oldTConfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), oldTConfig)

		exceptedBeforeCreateNewLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      ServerName,
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedBeforeCreateNewLabels, oldTConfig.Labels))

		newTConfigLayout := &crdV1Beta2.TConfig{
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

		newTConfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), newTConfigLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		exceptedNewTConfigLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      ServerName,
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedNewTConfigLabels, newTConfig.Labels))

		exceptedAfterCreateNewLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      ServerName,
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "false",
		}
		time.Sleep(time.Second * 1)
		oldTConfig, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), oldTConfig)
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(exceptedAfterCreateNewLabels, oldTConfig.Labels))

		err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Delete(context.TODO(), NewResourceName, k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)

		time.Sleep(time.Second * 1)
		oldTConfig, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
	})
})

var _ = ginkgo.Describe("test create slave config before master", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
	}

	s := scaffold.NewScaffold(opts)

	appLevelSlaveLayout := &crdV1Beta2.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      "app.config",
			Namespace: s.Namespace,
		},
		App:           "Test",
		PodSeq:        "1",
		ConfigName:    "app.config",
		ConfigContent: "I am app level",
		Activated:     true,
		UpdateTime:    k8sMetaV1.Now(),
	}

	serverLevelSlaveLayout := &crdV1Beta2.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      "server.config",
			Namespace: s.Namespace,
		},
		App:           "Test",
		Server:        "TestServer",
		PodSeq:        "1",
		ConfigName:    "server.config",
		ConfigContent: "I am server level",
		Activated:     true,
		UpdateTime:    k8sMetaV1.Now(),
	}
	var err error
	_, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), appLevelSlaveLayout, k8sMetaV1.CreateOptions{})
	assert.NotNil(ginkgo.GinkgoT(), err)

	_, err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), serverLevelSlaveLayout, k8sMetaV1.CreateOptions{})
	assert.NotNil(ginkgo.GinkgoT(), err)
})

var _ = ginkgo.Describe("test app level master/slaver config", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
	}

	s := scaffold.NewScaffold(opts)

	ServerApp := "Test"
	ConfigName := "app.conf"

	MasterResourceName := "app.config.master"

	MasterConfigContent := "Master Config Content"

	SlaverResourceName := "app.config.slave"
	SlaveConfigContent := "Slave Config Content"

	master := &crdV1Beta2.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      MasterResourceName,
			Namespace: s.Namespace,
		},
		App:           ServerApp,
		PodSeq:        "m",
		ConfigName:    ConfigName,
		ConfigContent: MasterConfigContent,
		Activated:     true,
		UpdateTime:    k8sMetaV1.Now(),
	}

	slave := &crdV1Beta2.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      SlaverResourceName,
			Namespace: s.Namespace,
		},
		App:           ServerApp,
		PodSeq:        "1",
		ConfigName:    ConfigName,
		ConfigContent: SlaveConfigContent,
		Activated:     true,
		UpdateTime:    k8sMetaV1.Now(),
	}

	ginkgo.BeforeEach(func() {
		masterTConfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), master, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), masterTConfig)

		masterConfigExceptedLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      "",
			crdMeta.TConfigPodSeqLabel:    "m",
			crdMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(masterConfigExceptedLabels, masterTConfig.Labels))

		slaveTConfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), slave, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), slaveTConfig)
		slaveConfigExceptedLabels := map[string]string{
			crdMeta.TServerAppLabel:       ServerApp,
			crdMeta.TServerNameLabel:      "",
			crdMeta.TConfigPodSeqLabel:    "1",
			crdMeta.TConfigActivatedLabel: "true",
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(slaveConfigExceptedLabels, slaveTConfig.Labels))
	})

	ginkgo.It("try delete app level tconfig", func() {
		err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Delete(context.TODO(), MasterResourceName, k8sMetaV1.DeleteOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)

		err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Delete(context.TODO(), SlaverResourceName, k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(time.Second)

		err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Delete(context.TODO(), MasterResourceName, k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
	})
})

var _ = ginkgo.Describe("test server level master/slaver config", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
	}

	s := scaffold.NewScaffold(opts)

	ServerApp := "Test"
	ServerName := "TestServer"
	ConfigName := "server.conf"

	MasterResourceName := "server.config.master"

	MasterConfigContent := "Master Config Content"

	SlaverResourceName := "server.config.slave"
	SlaveConfigContent := "Slave Config Content"

	master := &crdV1Beta2.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      MasterResourceName,
			Namespace: s.Namespace,
		},
		App:           ServerApp,
		Server:        ServerName,
		PodSeq:        "m",
		ConfigName:    ConfigName,
		ConfigContent: MasterConfigContent,
		Activated:     true,
		UpdateTime:    k8sMetaV1.Now(),
	}

	slave := &crdV1Beta2.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      SlaverResourceName,
			Namespace: s.Namespace,
		},
		App:           ServerApp,
		Server:        ServerName,
		PodSeq:        "1",
		ConfigName:    ConfigName,
		ConfigContent: SlaveConfigContent,
		Activated:     true,
		UpdateTime:    k8sMetaV1.Now(),
	}

	ginkgo.BeforeEach(func() {
		masterTConfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), master, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), masterTConfig)

		slaveTConfig, err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Create(context.TODO(), slave, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), slaveTConfig)
	})

	ginkgo.It("try delete server level master/slave tconfig", func() {
		err := s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Delete(context.TODO(), MasterResourceName, k8sMetaV1.DeleteOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)

		err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Delete(context.TODO(), SlaverResourceName, k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(time.Second)

		err = s.CRDClient.CrdV1beta1().TConfigs(s.Namespace).Delete(context.TODO(), MasterResourceName, k8sMetaV1.DeleteOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
	})
})
