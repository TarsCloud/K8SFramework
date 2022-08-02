package e2e

import (
	"context"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/propertyf"
	"github.com/TarsCloud/TarsGo/tars/util/conf"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/stretchr/testify/assert"
	tarsCrdV1Beta3 "k8s.tars.io/crd/v1beta3"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"strings"
	"testing"
	"time"
)

const propertyObjId = "tars.tarsproperty.PropertyObj"

func TestProperty(t *testing.T) {

	var propertyProxy = new(propertyf.PropertyF)

	var r *resources.Resources
	var esClient *elasticsearch.Client
	var propertyIndexPre string

	var MName = "E2E.Master" + RandStringRunes(5)
	var PName = "E2E.Property" + RandStringRunes(12)

	feature := features.New("Testing "+propertyObjId).WithLabel("crd-version", "v1beta3").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			r, _ = resources.New(config.Client().RESTConfig())
			_ = tarsCrdV1Beta3.AddToScheme(r.GetScheme())

			tt := &tarsCrdV1Beta3.TTemplate{}
			err := r.Get(ctx, "tars.es", namespace, tt)
			assert.Nil(t, err, "unexpected error")

			esTT := conf.New()
			err = esTT.InitFromString(tt.Spec.Content)
			assert.Nil(t, err, "unexpected error")

			protocol := esTT.GetStringWithDef("/tars/<protocol>", "http")

			nodes := esTT.GetDomainLine("/tars/elk/nodes")
			assert.NotNil(t, nodes, "get empty elasticsearch nodes")

			propertyIndexPre = esTT.GetString("/tars/elk/indexpre<property>")

			esClient, _ = elasticsearch.NewClient(elasticsearch.Config{
				Addresses: func() []string {
					var address []string
					for _, n := range nodes {
						address = append(address, protocol+"://"+n)
					}
					return address
				}(),
			})

			comm.StringToProxy(propertyObjId, propertyProxy)
			return ctx
		}).
		Assess("call rpc ReportPropMsg", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			msg := map[propertyf.StatPropMsgHead]propertyf.StatPropMsgBody{
				{
					ModuleName:   MName,
					PropertyName: PName,
				}: {
					VInfo: []propertyf.StatPropInfo{
						{
							Policy: "Avg",
							Value:  "50",
						},
						{
							Policy: "Sum",
							Value:  "50",
						},
					},
				},
			}
			ret, err := propertyProxy.ReportPropMsg(msg)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret: %d", ret)

			time.Sleep(75 * time.Second)

			query := `{"query":
							{"bool":{"must":[
										{"wildcard":{"master_name":"_MASTER_"}},
										{"wildcard":{"property_name":"_PROPERTY_"}}
                                        ]
                                     }
                            }
                      }`
			query = strings.ReplaceAll(query, "_MASTER_", MName)
			query = strings.ReplaceAll(query, "_PROPERTY_", PName)

			index := propertyIndexPre + time.Now().Format("20060102")
			sources, err := QueryES(esClient, index, query)
			assert.Nil(t, err)
			assert.Equal(t, len(sources), 2)
			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}
