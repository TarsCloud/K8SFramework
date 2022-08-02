package e2e

import (
	"context"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/endpointf"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/queryf"
	"github.com/stretchr/testify/assert"
	tarsCrdV1Beta3 "k8s.tars.io/crd/v1beta3"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

const queryObjId = "tars.tarsregistry.QueryObj"

func TestQueryObj(t *testing.T) {

	var queryProxy *queryf.QueryF
	var r *resources.Resources

	feature := features.New("Testing "+queryObjId).WithLabel("crd-version", "v1beta3").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			r, _ = resources.New(config.Client().RESTConfig())
			_ = tarsCrdV1Beta3.AddToScheme(r.GetScheme())

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
			assert.Equal(t, int32(0), ret, "unexpected ret: %d", ret)

			var te tarsCrdV1Beta3.TEndpoint
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
			var servant *tarsCrdV1Beta3.TServerServant
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
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}
