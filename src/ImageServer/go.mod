module tarsimage

go 1.16

require (
	github.com/gorilla/mux v1.8.0
	k8s.io/api v0.20.15
	k8s.io/apimachinery v0.20.15
	k8s.tars.io v1.0.0
)

replace k8s.tars.io v1.0.0 => ./../k8s.tars.io
