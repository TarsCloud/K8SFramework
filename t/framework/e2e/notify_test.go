package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/notifyf"
	"github.com/TarsCloud/TarsGo/tars/util/conf"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/stretchr/testify/assert"
	tarsAppsV1Beta3 "k8s.tars.io/apps/v1beta3"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"strings"
	"testing"
	"time"
)

const notifyObjId = "tars.tarsnotify.NotifyObj"

func queryES(client *elasticsearch.Client, index string, query string) (map[string]map[string]interface{}, error) {
	size := 10000
	res, err := esapi.SearchRequest{Index: []string{index}, Body: bytes.NewReader([]byte(query)), Pretty: true, TrackTotalHits: true, Size: &size}.Do(context.TODO(), client)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var responses map[string]interface{}
	if err = json.NewDecoder(res.Body).Decode(&responses); err != nil {
		if err != nil {
			return nil, fmt.Errorf("unexpected decoder error: %s", err)
		}
	}

	sources := map[string]map[string]interface{}{}

	for _, hit := range responses["hits"].(map[string]interface{})["hits"].([]interface{}) {
		_id := hit.(map[string]interface{})["_id"].(string)
		_source := hit.(map[string]interface{})["_source"]
		if _source != nil {
			sources[_id] = _source.(map[string]interface{})
		}
	}
	return sources, nil
}

func TestNotifyObj(t *testing.T) {

	const TestServerApp = "E2ETest"
	const TestServerName = "E2EFooServer"

	var notifyProxy *notifyf.Notify
	var r *resources.Resources
	var esClient *elasticsearch.Client
	var notifyIndex string

	feature := features.New("Testing "+notifyObjId).WithLabel("crd-version", "v1beta3").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {

			r, _ = resources.New(config.Client().RESTConfig())
			_ = tarsAppsV1Beta3.AddToScheme(r.GetScheme())

			tt := &tarsAppsV1Beta3.TTemplate{}
			err := r.Get(ctx, "tars.es", namespace, tt)
			assert.Nil(t, err, "unexpected error")

			esTT := conf.New()
			err = esTT.InitFromString(tt.Spec.Content)
			assert.Nil(t, err, "unexpected error")

			protocol := esTT.GetStringWithDef("/tars/<protocol>", "http")

			nodes := esTT.GetDomainLine("/tars/elk/nodes")
			assert.NotNil(t, nodes, "get empty elasticsearch nodes")

			notifyIndex = esTT.GetString("/tars/elk/index<notify>")

			esClient, _ = elasticsearch.NewClient(elasticsearch.Config{
				Addresses: func() []string {
					var address []string
					for _, n := range nodes {
						address = append(address, protocol+"://"+n)
					}
					return address
				}(),
			})
			notifyProxy = new(notifyf.Notify)
			comm.StringToProxy(notifyObjId, notifyProxy)

			return ctx
		}).
		Assess("call rpc NotifyServer", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			content := fmt.Sprintf("this is e2e testing NotifyServer @%d", time.Now().UnixMilli())
			err := notifyProxy.NotifyServer(TestServerApp+"."+TestServerName, notifyf.NOTIFYLEVEL_NOTIFYERROR, content)
			assert.Nil(t, err, "unexpected rcp error")
			time.Sleep(1800 * time.Millisecond)

			query := `{"query":{"bool":{"must":[{"query_string":{"default_field":"message","query":"_CONTENT_"}}]}}}`
			query = strings.ReplaceAll(query, "_CONTENT_", content)
			sources, err := queryES(esClient, notifyIndex, query)
			assert.Nil(t, err, "")

			var catch = false
			for _, source := range sources {
				if source["message"].(string) == content {
					catch = true
					break
				}
			}
			assert.True(t, catch, "does not catch expected value: %s", content)
			return ctx
		}).
		Assess("call rpc reportServer", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			content := fmt.Sprintf("this is e2e testing reportServer @%d", time.Now().UnixMilli())
			err := notifyProxy.ReportServer(TestServerApp+"."+TestServerName, "123", content)
			assert.Nil(t, err, "unexpected rcp error")
			time.Sleep(1800 * time.Millisecond)

			query := `{"query":{"bool":{"must":[{"query_string":{"default_field":"message","query":"_CONTENT_"}}]}}}`
			query = strings.ReplaceAll(query, "_CONTENT_", content)
			sources, err := queryES(esClient, notifyIndex, query)
			assert.Nil(t, err, "")

			var catch = false
			for _, source := range sources {
				if source["message"].(string) == content {
					catch = true
					break
				}
			}
			assert.True(t, catch, "does not catch expected value: %s", content)
			return ctx
		}).
		Assess("call rpc reportNotifyInfo", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			content := fmt.Sprintf("this is e2e testing reportNotifyInfo @%d", time.Now().UnixMilli())
			info := &notifyf.ReportInfo{
				SApp:      TestServerApp,
				SServer:   TestServerName,
				ELevel:    notifyf.NOTIFYLEVEL_NOTIFYERROR,
				SMessage:  content,
				SThreadId: "12222222",
			}

			err := notifyProxy.ReportNotifyInfo(info)
			if err != nil {
				fmt.Printf("@4 :%s\n", err.Error())
			}
			assert.Nil(t, err, "unexpected rcp error")
			time.Sleep(1800 * time.Millisecond)
			query := `{"query":{"bool":{"must":[{"query_string":{"default_field":"message","query":"_CONTENT_"}}]}}}`
			query = strings.ReplaceAll(query, "_CONTENT_", content)
			sources, err := queryES(esClient, notifyIndex, query)
			assert.Nil(t, err, "")
			var catch = false
			for _, source := range sources {
				if source["message"].(string) == content {
					catch = true
					break
				}
			}
			assert.True(t, catch, "does not catch expected value: %s", content)
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			return ctx
		}).
		Feature()
	testenv.Test(t, feature)
}
