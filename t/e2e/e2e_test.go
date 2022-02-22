package e2e

import (
	"github.com/onsi/ginkgo"
	"testing"
)

func TestRunE2E(t *testing.T) {
	runE2E()
	ginkgo.RunSpecs(t, "tars k8s e2e test suites")
}
