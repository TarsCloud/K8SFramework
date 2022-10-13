module tarsagent

go 1.16

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/onsi/gomega v1.10.3 // indirect
	github.com/robfig/cron/v3 v3.0.1
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.20.15
	k8s.io/apimachinery v0.20.15
	k8s.io/client-go v0.20.15
	k8s.io/klog/v2 v2.5.0 // indirect
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	k8s.tars.io v1.0.0
)

replace k8s.tars.io v1.0.0 => ./../k8s.tars.io
