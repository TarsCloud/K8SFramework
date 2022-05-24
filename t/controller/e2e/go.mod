module e2e

go 1.14

require (
	github.com/gruntwork-io/terratest v0.40.0
	github.com/onsi/ginkgo v1.12.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	k8s.tars.io v0.0.1
)

replace k8s.tars.io v0.0.1 => ../../../src/k8s.tars.io/
