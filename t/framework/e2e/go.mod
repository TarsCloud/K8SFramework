module e2e

go 1.16

require (
	github.com/TarsCloud/TarsGo v1.3.5
	k8s.tars.io v0.0.1
	sigs.k8s.io/e2e-framework v0.0.7
)

require (
	github.com/elastic/go-elasticsearch/v7 v7.17.1
	github.com/stretchr/testify v1.7.1
	k8s.io/apimachinery v0.24.0
	sigs.k8s.io/controller-runtime v0.12.1 // indirect
)

replace k8s.tars.io v0.0.1 => ../../../src/k8s.tars.io/
