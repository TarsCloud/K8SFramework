module tarskaniko

go 1.16

require (
	github.com/GoogleContainerTools/kaniko v1.8.1
	github.com/containerd/containerd v1.6.2
	github.com/google/go-containerregistry v0.8.1-0.20220214202839-625fe7b4276a
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	k8s.tars.io v1.0.0
)

replace (
	github.com/moby/buildkit => github.com/moby/buildkit v0.8.3
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc92
	github.com/tonistiigi/fsutil => github.com/tonistiigi/fsutil v0.0.0-20201103201449-0834f99b7b85
	k8s.tars.io v1.0.0 => ./../k8s.tars.io
)
