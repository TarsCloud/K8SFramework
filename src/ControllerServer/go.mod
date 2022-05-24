module tarscontroller

go 1.15

require (
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	k8s.io/api v0.18.19
	k8s.io/apiextensions-apiserver v0.18.19
	k8s.io/apimachinery v0.18.19
	k8s.io/client-go v0.18.19
	k8s.io/utils v0.0.0-20200324210504-a9aa75ae1b89
	k8s.tars.io v0.0.1
)

replace k8s.tars.io v0.0.1 => ../k8s.tars.io/
