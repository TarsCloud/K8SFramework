module tarsagent

go 1.14

require (
	github.com/containerd/containerd v1.5.2 // indirect
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.8.0 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887 // indirect
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	k8s.io/klog/v2 v2.5.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	k8s.tars.io v1.0.0
)

replace k8s.tars.io v1.0.0 => ./../k8s.tars.io
