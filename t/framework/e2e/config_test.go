package e2e

import (
	"context"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/configf"
	"github.com/stretchr/testify/assert"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1Beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"strings"
	"testing"
	"time"
)

const configObjId = "tars.tarsconfig.ConfigObj"

func TestAppLevelTConfig(t *testing.T) {

	var layout1 = &tarsCrdV1Beta3.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: "v1",
		},
		App:           "Test",
		Server:        "",
		PodSeq:        "m",
		ConfigName:    "example.conf",
		Version:       "",
		ConfigContent: "Example Context V1",
		UpdateTime:    k8sMetaV1.Now(),
		Activated:     false,
	}

	var layout2 = &tarsCrdV1Beta3.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: "v2",
		},
		App:           "Test",
		Server:        "",
		PodSeq:        "m",
		ConfigName:    "example.conf",
		ConfigContent: "Example Context V2",
		Activated:     true,
	}

	var configProxy = new(configf.Config)
	comm.StringToProxy(configObjId, configProxy)

	var r *resources.Resources
	var v1Tconfig *tarsCrdV1Beta3.TConfig
	var v2Tconfig *tarsCrdV1Beta3.TConfig

	feature := features.New("Testing "+configObjId).WithLabel("crd-version", "v1beta3").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			r, _ = resources.New(config.Client().RESTConfig())
			_ = tarsCrdV1Beta3.AddToScheme(r.GetScheme())
			v1Tconfig = &tarsCrdV1Beta3.TConfig{}
			err := decoder.DecodeString(ObjLayoutToString(layout1, namespace), v1Tconfig)
			assert.Nil(t, err, "decode tconfig layout failed")

			v2Tconfig = &tarsCrdV1Beta3.TConfig{}
			err = decoder.DecodeString(ObjLayoutToString(layout2, namespace), v2Tconfig)
			assert.Nil(t, err, "decode tconfig layout failed")

			return ctx
		}).
		Assess("compare v1Tconfig and v2Tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			assert.NotEqual(t, v1Tconfig.Name, v2Tconfig.Name)
			assert.NotEqual(t, v1Tconfig.ConfigContent, v2Tconfig.ConfigContent)

			assert.Equal(t, v1Tconfig.App, v2Tconfig.App)
			assert.Equal(t, v1Tconfig.Server, v2Tconfig.Server)
			assert.Equal(t, v1Tconfig.ConfigName, v2Tconfig.ConfigName)
			assert.Equal(t, v1Tconfig.PodSeq, v2Tconfig.PodSeq)
			return ctx
		}).
		Assess("create v1Tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			err := r.Create(ctx, v1Tconfig)
			assert.Nil(t, err, "create tconfig failed")
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Assess("call rpc listConfig before activate tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var names []string
			ret, err := configProxy.ListConfig("Test", "", &names)
			assert.Nil(t, err, "unexpected rpc error")
			assert.Equal(t, int32(0), ret, "unexpected rpc ret")
			assert.Equal(t, 0, len(names), "unexpected lengths of tconfig")
			return ctx
		}).
		Assess("activate tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			patch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/activated",
					Value: true,
				},
			}
			patchBS, _ := json.Marshal(patch)
			err := r.Patch(ctx, v1Tconfig, k8s.Patch{
				PatchType: types.JSONPatchType,
				Data:      patchBS,
			})
			assert.Nil(t, err, "unexpected error")
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Assess("call rpc listConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var names []string
			ret, err := configProxy.ListConfig("Test", "", &names)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret code: %d", ret)
			assert.Equal(t, 1, len(names), "unexpected lengths of tconfig: %d", len(names))
			assert.Equal(t, v1Tconfig.ConfigName, names[0], "unexpected lengths of tconfig: %s", len(names))
			return ctx
		}).
		Assess("rpc getConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var content string
			ret, err := configProxy.LoadConfig(v1Tconfig.App, v1Tconfig.Server, v1Tconfig.ConfigName, &content)
			assert.Nil(t, err, "unexpected rpc error")
			assert.Equal(t, int32(0), ret, "unexpected rpc ret")
			assert.Equal(t, v1Tconfig.ConfigContent, content, "unexpected tconfig content")
			return ctx
		}).
		Assess("create v2 tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			err := r.Create(ctx, v2Tconfig)
			assert.Nil(t, err, "create tconfig failed")
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Assess("call rpc listConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var names []string
			ret, err := configProxy.ListConfig("Test", "", &names)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret code: %d", ret)
			assert.Equal(t, 1, len(names), "unexpected lengths of tconfig: %d", len(names))
			assert.Equal(t, v2Tconfig.ConfigName, names[0], "unexpected lengths of tconfig: %s", len(names))
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Assess("rpc getConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var content string
			ret, err := configProxy.LoadConfig(v2Tconfig.App, v1Tconfig.Server, v1Tconfig.ConfigName, &content)
			assert.Nil(t, err, "unexpected rpc error")
			assert.Equal(t, int32(0), ret, "unexpected rpc ret")
			assert.Equal(t, v2Tconfig.ConfigContent, content, "unexpected tconfig content")
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			time.Sleep(50 * time.Second) //skip tconfig deletion guard time
			_ = r.Delete(ctx, v1Tconfig)
			_ = r.Delete(ctx, v2Tconfig)
			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}

func TestServerLevelTConfig(t *testing.T) {

	var layout1 = &tarsCrdV1Beta3.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: "v1",
		},
		App:           "Test",
		Server:        "Foo",
		PodSeq:        "m",
		ConfigName:    "example.conf",
		ConfigContent: "Example Context V1",
		UpdateTime:    k8sMetaV1.Now(),
		Activated:     false,
	}

	var layout2 = &tarsCrdV1Beta3.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: "v2",
		},
		App:           "Test",
		Server:        "Foo",
		PodSeq:        "m",
		ConfigName:    "example.conf",
		ConfigContent: "Example Context V2",
		Activated:     true,
	}

	var layoutSlave = &tarsCrdV1Beta3.TConfig{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: "slave",
		},
		App:           "Test",
		Server:        "Foo",
		PodSeq:        "1",
		ConfigName:    "example.conf",
		ConfigContent: "Slave Context",
		Activated:     true,
	}

	var configProxy = new(configf.Config)
	comm.StringToProxy(configObjId, configProxy)

	var r *resources.Resources
	var v1Tconfig *tarsCrdV1Beta3.TConfig
	var v2Tconfig *tarsCrdV1Beta3.TConfig
	var slaveTconfig *tarsCrdV1Beta3.TConfig

	feature := features.New("Testing "+configObjId).WithLabel("crd-version", "v1beta3").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			r, _ = resources.New(config.Client().RESTConfig())
			_ = tarsCrdV1Beta3.AddToScheme(r.GetScheme())
			v1Tconfig = &tarsCrdV1Beta3.TConfig{}
			err := decoder.DecodeString(ObjLayoutToString(layout1, namespace), v1Tconfig)
			assert.Nil(t, err, "decode tconfig layout failed")

			v2Tconfig = &tarsCrdV1Beta3.TConfig{}
			err = decoder.DecodeString(ObjLayoutToString(layout2, namespace), v2Tconfig)
			assert.Nil(t, err, "decode tconfig layout failed")

			slaveTconfig = &tarsCrdV1Beta3.TConfig{}
			err = decoder.DecodeString(ObjLayoutToString(layoutSlave, namespace), slaveTconfig)
			assert.Nil(t, err, "decode tconfig layout failed")

			return ctx
		}).
		Assess("compare v1Tconfig and v2Tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			assert.NotEqual(t, v1Tconfig.Name, v2Tconfig.Name)
			assert.NotEqual(t, v1Tconfig.ConfigContent, v2Tconfig.ConfigContent)

			assert.Equal(t, v1Tconfig.App, v2Tconfig.App)
			assert.Equal(t, v1Tconfig.Server, v2Tconfig.Server)
			assert.Equal(t, v1Tconfig.ConfigName, v2Tconfig.ConfigName)
			assert.Equal(t, v1Tconfig.PodSeq, v2Tconfig.PodSeq)
			return ctx
		}).
		Assess("create v1Tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			err := r.Create(ctx, v1Tconfig)
			assert.Nil(t, err, "create tconfig failed")
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Assess("call rpc listConfig before activate tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var names []string
			ret, err := configProxy.ListConfig(v1Tconfig.App, v1Tconfig.Server, &names)
			assert.Nil(t, err, "unexpected rpc error")
			assert.Equal(t, int32(0), ret, "unexpected rpc ret")
			assert.Equal(t, 0, len(names), "unexpected lengths of tconfig")
			return ctx
		}).
		Assess("activate tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			patch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/activated",
					Value: true,
				},
			}
			patchBS, _ := json.Marshal(patch)
			err := r.Patch(ctx, v1Tconfig, k8s.Patch{
				PatchType: types.JSONPatchType,
				Data:      patchBS,
			})
			assert.Nil(t, err, "unexpected error")
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Assess("call rpc listConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var names []string
			ret, err := configProxy.ListConfig(v1Tconfig.App, v1Tconfig.Server, &names)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret code: %d", ret)
			assert.Equal(t, 1, len(names), "unexpected lengths of tconfig: %d", len(names))
			assert.Equal(t, v1Tconfig.ConfigName, names[0], "unexpected lengths of tconfig: %s", len(names))
			return ctx
		}).
		Assess("rpc getConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var content string
			ret, err := configProxy.LoadConfig(v1Tconfig.App, v1Tconfig.Server, v1Tconfig.ConfigName, &content)
			assert.Nil(t, err, "unexpected rpc error")
			assert.Equal(t, int32(0), ret, "unexpected rpc ret")
			assert.Equal(t, v1Tconfig.ConfigContent, content, "unexpected tconfig content")
			return ctx
		}).
		Assess("create v2 tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			err := r.Create(ctx, v2Tconfig)
			assert.Nil(t, err, "create tconfig failed")
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Assess("call rpc listConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var names []string
			ret, err := configProxy.ListConfig(v2Tconfig.App, v2Tconfig.Server, &names)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret code: %d", ret)
			assert.Equal(t, 1, len(names), "unexpected lengths of tconfig: %d", len(names))
			assert.Equal(t, v2Tconfig.ConfigName, names[0], "unexpected lengths of tconfig: %s", len(names))
			return ctx
		}).
		Assess("rpc getConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var content string
			ret, err := configProxy.LoadConfig(v2Tconfig.App, v2Tconfig.Server, v2Tconfig.ConfigName, &content)
			assert.Nil(t, err, "unexpected rpc error")
			assert.Equal(t, int32(0), ret, "unexpected rpc ret")
			assert.Equal(t, v2Tconfig.ConfigContent, content, "unexpected tconfig content")
			return ctx
		}).
		Assess("create slave tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			err := r.Create(ctx, slaveTconfig)
			assert.Nil(t, err, "create tconfig failed")
			time.Sleep(time.Second * 2)
			return ctx
		}).
		Assess("call rpc listConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var names []string
			ret, err := configProxy.ListConfig(slaveTconfig.App, slaveTconfig.Server, &names)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret code: %d", ret)
			assert.Equal(t, 1, len(names), "unexpected lengths of tconfig: %d", len(names))
			assert.Equal(t, slaveTconfig.ConfigName, names[0], "unexpected lengths of tconfig: %s", len(names))
			return ctx
		}).
		Assess("rpc getConfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			var content string
			ret, err := configProxy.LoadConfigByHost(slaveTconfig.App+"."+slaveTconfig.Server, slaveTconfig.ConfigName, strings.ToLower(slaveTconfig.App+"-"+slaveTconfig.Server+"-"+slaveTconfig.PodSeq), &content)
			assert.Nil(t, err, "unexpected rpc error")
			assert.Equal(t, int32(0), ret, "unexpected rpc ret")
			assert.Equal(t, v2Tconfig.ConfigContent+"\r\n\r\n"+slaveTconfig.ConfigContent, content, "unexpected tconfig content")
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			time.Sleep(50 * time.Second) //skip tconfig deletion guard time
			_ = r.Delete(ctx, slaveTconfig)
			_ = r.Delete(ctx, v2Tconfig)
			_ = r.Delete(ctx, v1Tconfig)
			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}
