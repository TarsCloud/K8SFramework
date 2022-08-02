package e2e

import (
	"context"
	"github.com/TarsCloud/TarsGo/tars/util/conf"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/stretchr/testify/assert"
	tarsCrdV1Beta3 "k8s.tars.io/crd/v1beta3"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"strings"
	"testing"
)

func TestKEvent(t *testing.T) {

	var r *resources.Resources
	var esClient *elasticsearch.Client
	var eventIndex string

	feature := features.New("Testing "+"KEvent").
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

			eventIndex = esTT.GetString("/tars/elk/index<kevent>")

			esClient, _ = elasticsearch.NewClient(elasticsearch.Config{
				Addresses: func() []string {
					var address []string
					for _, n := range nodes {
						address = append(address, protocol+"://"+n)
					}
					return address
				}(),
			})

			return ctx
		}).
		Assess("check Statefulset Events", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			query := `{
                        "query":{
                            "bool":{
                            	"must":[
                                	{"wildcard":{"reason":"SuccessfulCreate"}},
                      		    	{"wildcard":{"involvedObject.kind":"Statefulset"}},
                                	{"wildcard":{"involvedObject.namespace":"_NAMESPACE_"}},
								    {"match":{"involvedObject.name":"tars-"}}
                                ]
                           }
                       },
                       "sort":[{"@timestamp":"desc"},{"_id":"desc"}]
                     }`
			query = strings.ReplaceAll(query, "_NAMESPACE_", namespace)
			sources, err := QueryES(esClient, eventIndex, query)
			assert.Nil(t, err, "")
			expectedStatefulsets := map[string]interface{}{
				"tars-elasticsearch":     nil,
				"tars-tarsconfig":        nil,
				"tars-tarsimage":         nil,
				"tars-tarskevent":        nil,
				"tars-tarslog":           nil,
				"tars-tarsnotify":        nil,
				"tars-tarsproperty":      nil,
				"tars-tarsqueryproperty": nil,
				"tars-tarsquerystat":     nil,
				"tars-tarsregistry":      nil,
				"tars-tarsstat":          nil,
				"tars-tarsweb":           nil,
			}
			for _, source := range sources {
				involvedObject := source["involvedObject"].(map[string]interface{})
				name, ok := involvedObject["name"].(string)
				if ok {
					delete(expectedStatefulsets, name)
				}
				if len(expectedStatefulsets) == 0 {
					break
				}
			}
			assert.Zero(t, len(expectedStatefulsets), "does not catch expected statefulset")
			return ctx
		}).
		Assess("check Pod Events", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			query := `{
                        "query":{
                            "bool":{
                            	"must":[
                                	{"wildcard":{"involvedObject.kind":"Pod"}},
                      		    	{"wildcard":{"reason":"Started"}},
                                	{"wildcard":{"involvedObject.namespace":"_NAMESPACE_"}},
									{"match":{"involvedObject.name":"tars-"}}
                                ]
                           }
                       },
                       "sort":[{"@timestamp":"desc"},{"_id":"desc"}]
                     }`
			query = strings.ReplaceAll(query, "_NAMESPACE_", namespace)
			sources, err := QueryES(esClient, eventIndex, query)
			assert.Nil(t, err, "")
			expectedPods := map[string]interface{}{
				"tars-elasticsearch-0":     nil,
				"tars-tarsconfig-0":        nil,
				"tars-tarsimage-0":         nil,
				"tars-tarskevent-0":        nil,
				"tars-tarslog-0":           nil,
				"tars-tarsnotify-0":        nil,
				"tars-tarsproperty-0":      nil,
				"tars-tarsqueryproperty-0": nil,
				"tars-tarsquerystat-0":     nil,
				"tars-tarsregistry-0":      nil,
				"tars-tarsregistry-1":      nil,
				"tars-tarsstat-0":          nil,
				"tars-tarsweb-0":           nil,
			}
			for _, source := range sources {
				involvedObject := source["involvedObject"].(map[string]interface{})
				name, ok := involvedObject["name"].(string)
				if ok {
					delete(expectedPods, name)
				}
				if len(expectedPods) == 0 {
					break
				}
			}
			assert.Zero(t, len(expectedPods), "does not catch expected pods")
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}
