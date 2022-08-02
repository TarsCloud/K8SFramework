package e2e

import (
	"context"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/statf"
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

const statObjId = "tars.tarsstat.StatObj"

func TestStat(t *testing.T) {

	var statProxy = new(statf.StatF)

	var r *resources.Resources
	var esClient *elasticsearch.Client
	var statIndexPre string

	var MName = "E2E.Master" + RandStringRunes(5)
	var SName = "E2E.Slave" + RandStringRunes(5)
	var Interface = "E2EInterface" + RandStringRunes(5)
	const SlavePort = 9200

	feature := features.New("Testing "+statObjId).WithLabel("crd-version", "v1beta3").
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

			statIndexPre = esTT.GetString("/tars/elk/indexpre<stat>")

			esClient, _ = elasticsearch.NewClient(elasticsearch.Config{
				Addresses: func() []string {
					var address []string
					for _, n := range nodes {
						address = append(address, protocol+"://"+n)
					}
					return address
				}(),
			})

			comm.StringToProxy(statObjId, statProxy)
			return ctx
		}).
		Assess("call rpc ReportMicMsg", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			msg := map[statf.StatMicMsgHead]statf.StatMicMsgBody{
				{
					MasterName:    MName,
					SlaveName:     SName,
					InterfaceName: Interface,
					MasterIp:      "",
					SlaveIp:       "",
					SlavePort:     SlavePort,
					ReturnValue:   0,
					SlaveSetName:  "",
					SlaveSetArea:  "",
					SlaveSetID:    "",
					TarsVersion:   "",
				}: {
					Count:        1000,
					TimeoutCount: 1000,
					ExecCount:    1000,
					IntervalCount: map[int32]int32{
						5:    0,
						10:   1,
						50:   1,
						500:  1,
						1000: 2,
						2000: 3,
						3000: 5,
					},
					TotalRspTime: 100,
					MaxRspTime:   100,
					MinRspTime:   10,
				},
			}
			ret, err := statProxy.ReportMicMsg(msg, true)
			assert.Nil(t, err, "unexpected rcp error")
			assert.Equal(t, int32(0), ret, "unexpected ret: %d", ret)

			time.Sleep(90 * time.Second)

			query := `{"query":
							{"bool":{"must":[
										{"wildcard":{"master_name":"_MASTER_"}},
										{"wildcard":{"slave_name":"_SLAVE_"}},
										{"wildcard":{"interface_name":"_INTERFACE_"}}
                                        ]
                                     }
                            }
                      }`
			query = strings.ReplaceAll(query, "_MASTER_", MName)
			query = strings.ReplaceAll(query, "_SLAVE_", SName)
			query = strings.ReplaceAll(query, "_INTERFACE_", Interface)

			index := statIndexPre + time.Now().Format("20060102")
			sources, err := QueryES(esClient, index, query)
			assert.Nil(t, err)
			assert.GreaterOrEqual(t, len(sources), 1)

			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}
