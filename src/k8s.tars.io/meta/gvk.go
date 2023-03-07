package meta

const (
	V1beta1 = "v1beta1"
	V1beta2 = "v1beta2"
	V1beta3 = "v1beta3"
)

const (
	TarsGroup            = "k8s.tars.io"
	TarsGroupVersionV1B1 = "k8s.tars.io/v1beta1"
	TarsGroupVersionV1B2 = "k8s.tars.io/v1beta2"
	TarsGroupVersionV1B3 = "k8s.tars.io/v1beta3"
)

const (
	KNodeKind                  = "Node"
	KServiceKind               = "Service"
	KPodKind                   = "Pod"
	KPersistentVolumeClaimKind = "PersistentVolumeClaim"
	KStatefulSetKind           = "StatefulSet"
	KDaemonSetKind             = "Daemonset"
)

const (
	TServerKind          = "TServer"
	TImageKind           = "TImage"
	TConfigKind          = "TConfig"
	TAccountKind         = "TAccount"
	TTemplateKind        = "Template"
	TEndpointKind        = "TEndpoint"
	TTreeKind            = "TTree"
	TExitedRecordKind    = "TExitedRecord"
	TFrameworkConfigKind = "TFrameworkConfig"
)
