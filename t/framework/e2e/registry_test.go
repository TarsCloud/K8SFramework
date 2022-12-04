package e2e

import (
	"context"
	"fmt"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/endpointf"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/queryf"
	"github.com/stretchr/testify/assert"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1Beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsTool "k8s.tars.io/tool"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
	"time"
)

const queryObjId = "tars.tarsregistry.QueryObj"

func TestQueryObj(t *testing.T) {

	var queryProxy *queryf.QueryF
	var r *resources.Resources

	feature := features.New("Testing "+queryObjId).WithLabel("crd-version", "v1beta3").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			r, _ = resources.New(config.Client().RESTConfig())
			_ = tarsV1Beta3.AddToScheme(r.GetScheme())

			queryProxy = new(queryf.QueryF)
			comm.StringToProxy(queryObjId, queryProxy)

			return ctx
		}).
		Assess("call rpc findEndpoint", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			const configID = "tars.tarsconfig.ConfigObj"
			var activateEP []endpointf.EndpointF
			var inactivateEP []endpointf.EndpointF
			ret, err := queryProxy.FindObjectById4All(configID, &activateEP, &inactivateEP)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret code: %d", ret)

			var te tarsV1Beta3.TEndpoint
			err = r.Get(ctx, "tars-tarsconfig", namespace, &te)
			assert.Nil(t, err, "unexpected get error")
			assert.NotNil(t, te)

			activatePodInTe := map[string]interface{}{}
			inactivatePodInTe := map[string]interface{}{}

			for _, s := range te.Status.PodStatus {
				if s.PresentState == "Active" && s.SettingState == "Active" {
					activatePodInTe[s.Name+".tars-tarsconfig"] = nil
				} else {
					inactivatePodInTe[s.Name+".tars-tarsconfig"] = nil
				}
			}
			var servant *tarsV1Beta3.TServerServant
			for _, s := range te.Spec.Tars.Servants {
				if s.Name == "ConfigObj" {
					servant = s
					break
				}
			}

			assert.NotNil(t, servant)
			assert.Equal(t, len(activatePodInTe), len(activateEP))
			for _, ep := range activateEP {
				_, ok := activatePodInTe[ep.Host]
				assert.True(t, ok, "%s should in activatePodInTe", ep.Host)
				assert.Equal(t, servant.Port, ep.Port)
				assert.Equal(t, servant.Timeout, ep.Timeout)
				if servant.IsTcp {
					assert.Equal(t, int32(1), ep.Istcp)
				} else {
					assert.Equal(t, int32(0), ep.Istcp)
				}
			}

			assert.Equal(t, len(inactivatePodInTe), len(inactivateEP))

			for _, ep := range inactivateEP {
				_, ok := inactivatePodInTe[ep.Host]
				assert.True(t, ok, "%s should in inactivatePodInTe", ep.Host)
				assert.Equal(t, servant.Port, ep.Port)
				assert.Equal(t, servant.Timeout, ep.Timeout)
				if servant.IsTcp {
					assert.Equal(t, int32(1), ep.Istcp)
				} else {
					assert.Equal(t, int32(0), ep.Istcp)
				}
			}
			return ctx
		}).
		Assess("test upChain", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {

			var falseValue = false
			var trueValue = true

			upChain := map[string][]tarsV1Beta3.TFrameworkTarsEndpoint{
				"default": {
					{
						Host: "default.1",
						Port: 3333,
					},
					{
						Host:    "default.2",
						Port:    3333,
						Timeout: 60000,
						IsTcp:   &falseValue,
					},
				},
				"e2e.Test.TestObj": {
					{
						Host:  "test.1",
						Port:  44444,
						IsTcp: &falseValue,
					},
					{
						Host:    "test.2",
						Port:    44444,
						Timeout: 5000,
						IsTcp:   &trueValue,
					},
				},
			}
			tfc := &tarsV1Beta3.TFrameworkConfig{}
			patch := tarsTool.JsonPatch{
				{
					OP:    tarsTool.JsonPatchReplace,
					Path:  "/upChain",
					Value: upChain,
				},
			}

			tfc = &tarsV1Beta3.TFrameworkConfig{
				ObjectMeta: k8sMetaV1.ObjectMeta{
					Name:      "tars-framework",
					Namespace: scaffold.Namespace,
				},
			}
			tafLayout := &tarsV1Beta3.TFrameworkConfig{}
			err := decoder.DecodeString(ObjLayoutToString(tafLayout, namespace), tfc)
			patchBS, _ := json.Marshal(patch)
			err = r.Patch(ctx, tfc, k8s.Patch{
				PatchType: types.JSONPatchType,
				Data:      patchBS,
			})
			assert.Nil(t, err, "unexpected error")
			time.Sleep(time.Second * 2)

			var activateEP []endpointf.EndpointF
			var inactivateEP []endpointf.EndpointF

			const DefaultTimeoutValue int32 = 6000
			const DefaultISTcp int32 = 1
			var ret int32

			ret, err = queryProxy.FindObjectById4All("I.Am.NotExistObj", &activateEP, &inactivateEP)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret code: %d", ret)
			assert.Equal(t, 2, len(activateEP))
			assert.Equal(t, 0, len(inactivateEP))

			for _, v := range activateEP {
				if v.Host == "default.1" {
					assert.Equal(t, int32(3333), v.Port)
					assert.Equal(t, DefaultTimeoutValue, v.Timeout)
					assert.Equal(t, DefaultISTcp, v.Istcp)
				}

				if v.Host == "default.2" {
					assert.Equal(t, int32(3333), v.Port)
					assert.Equal(t, int32(60000), v.Timeout)
					assert.Equal(t, int32(0), v.Istcp)
				}
				assert.Error(t, fmt.Errorf("unexpected endpoints host: %s", v.Host))
			}

			ret, err = queryProxy.FindObjectById4All("e2e.Test.TestObj", &activateEP, &inactivateEP)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret code: %d", ret)
			assert.Equal(t, 2, len(activateEP))
			assert.Equal(t, 0, len(inactivateEP))

			for _, v := range activateEP {
				if v.Host == "test.1" {
					assert.Equal(t, int32(44444), v.Port)
					assert.Equal(t, DefaultTimeoutValue, v.Timeout)
					assert.Equal(t, int32(0), v.Istcp)
				}

				if v.Host == "test.2" {
					assert.Equal(t, int32(44444), v.Port)
					assert.Equal(t, int32(5000), v.Timeout)
					assert.Equal(t, int32(1), v.Istcp)
				}
				assert.Error(t, fmt.Errorf("unexpected endpoints host: %s", v.Host))
			}

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}
