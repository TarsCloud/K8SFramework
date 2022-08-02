package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/TarsCloud/TarsGo/tars"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"math/rand"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

type Scaffold struct {
	TestEnv   env.Environment
	Comm      *tars.Communicator
	Namespace string
}

var scaffold = Scaffold{
	TestEnv: nil,
	Comm:    nil,
}

func GetScaffold() *Scaffold {
	return &scaffold
}

func QueryES(client *elasticsearch.Client, index string, query string) (map[string]map[string]interface{}, error) {
	res, err := esapi.SearchRequest{Index: []string{index}, Body: bytes.NewReader([]byte(query)), Pretty: true, TrackTotalHits: true}.Do(context.TODO(), client)
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

func RandStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
