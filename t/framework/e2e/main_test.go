/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"github.com/TarsCloud/TarsGo/tars"
	"os"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"testing"
)

var (
	testenv   env.Environment
	namespace = "tars"
	comm      *tars.Communicator
	locator   = "tars.tarsregistry.QueryObj@tcp -h tars-tarsregistry -p 17890 -t 3000"
)

func TestMain(m *testing.M) {
	comm = tars.NewCommunicator()
	comm.SetLocator(locator)
	testenv = env.New()

	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	testenv = env.NewWithConfig(cfg)
	testenv.BeforeEachFeature()
	os.Exit(testenv.Run(m))
}
