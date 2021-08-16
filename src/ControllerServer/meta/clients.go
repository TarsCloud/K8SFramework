package meta

import (
	"k8s.io/client-go/kubernetes"
	k8sMetadata "k8s.io/client-go/metadata"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
)

type Clients struct {
	K8sClient         kubernetes.Interface
	CrdClient         crdVersioned.Interface
	K8sMetadataClient k8sMetadata.Interface
}
