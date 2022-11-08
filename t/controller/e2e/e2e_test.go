package e2e

import (
	"github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/util/runtime"
	tarsRuntime "k8s.tars.io/runtime"
	"os/user"
	"path/filepath"
	"testing"
)

func TestRunE2E(t *testing.T) {
	runE2E()
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	runtime.Must(tarsRuntime.CreateContext("", filepath.Join(u.HomeDir, ".kube", "config")))
	ginkgo.RunSpecs(t, "tars k8s e2e test suites")
}
