package e2e

import (
	"context"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/statf"
	tarsV1Beta3 "k8s.tars.io/apis/tars/v1beta3"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

const statObjId = "tars.tarsstat.StatObj"

func TestStat(t *testing.T) {

	var statProxy = new(statf.StatF)
	comm.StringToProxy(statObjId, statProxy)

	var r *resources.Resources

	feature := features.New("Testing "+statObjId).WithLabel("crd-version", "v1beta3").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			r, _ = resources.New(config.Client().RESTConfig())
			_ = tarsV1Beta3.AddToScheme(r.GetScheme())

			return ctx
		}).
		Assess("compare v1Tconfig and v2Tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			return ctx
		}).
		Assess("create v1Tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			return ctx
		}).
		Assess("call rpc listConfig before activate tconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}
