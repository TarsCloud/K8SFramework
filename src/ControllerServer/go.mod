module tarscontroller

go 1.15

require (
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	k8s.io/api v0.20.15
	k8s.io/apiextensions-apiserver v0.20.15
	k8s.io/apimachinery v0.20.15
	k8s.io/client-go v0.20.15
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	k8s.tars.io v0.0.1
)

replace k8s.tars.io v0.0.1 => ../k8s.tars.io/
