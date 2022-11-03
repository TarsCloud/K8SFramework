package runtime

import (
	"k8s.io/client-go/kubernetes"
	k8sMetadata "k8s.io/client-go/metadata"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
)

type Client struct {
	K8sClient         kubernetes.Interface
	K8sMetadataClient k8sMetadata.Interface
	CrdClient         crdVersioned.Interface
}
