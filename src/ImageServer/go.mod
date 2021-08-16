module tarsimage

go 1.14

require (
	credentialprovider v1.0.0
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/Microsoft/hcsshim v0.8.14 // indirect
	github.com/containerd/containerd v1.4.3 // indirect
	github.com/containerd/continuity v0.0.0-20201208142359-180525291bb7 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v20.10.2+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/sirupsen/logrus v1.7.0 // indirect
	gotest.tools/v3 v3.0.3 // indirect
	k8s.io/api v0.18.19
	k8s.io/apimachinery v0.18.19
	k8s.io/client-go v0.18.19
	k8s.tars.io v1.0.0
)

replace k8s.tars.io v1.0.0 => ./../k8s.tars.io

replace credentialprovider v1.0.0 => ./credentialprovider
