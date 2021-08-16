package meta

const GarbageCollectorAccount = "system:serviceaccount:kube-system:generic-garbage-collector"

const (
	ResourceOutControlReason = "OutControl"

	ResourceDeleteReason = "DeleteError"

	ResourceGetReason = "GetError"
)

const (
	// ResourceOutControlError = "kind namespace/name already exists but not managed by namespace/name"
	ResourceOutControlError = "%s %s/%s already exists but not managed by %s/%s"

	// ResourceExistError = "kind namespace/name already exists"
	ResourceExistError = "%s %s/%s already exists"

	// ResourceNotExistError = "kind namespace/name already exists"
	ResourceNotExistError = "%s %s/%s not exists"

	// FiledImmutableError = "kind resource \"filed\" is immutable"
	FiledImmutableError = "%s resource filed \"%s\" is immutable"

	// ResourceDeleteError = "delete kind namespace/name err: errMsg"
	ResourceDeleteError = "delete %s %s/%s err: %s"

	// ResourceDeleteCollectionError ResourceDeleteError = "deleteCollection kind selector(labelSelector) err: errMsg"
	ResourceDeleteCollectionError = "deleteCollection %s selector(%s) err: %s"

	//ResourceGetError = "get kind namespace/name err: errMsg"
	ResourceGetError = "get %s %s/%s error: %s"

	//ResourceCreateError = "create kind namespace/name err: errMsg"
	ResourceCreateError = "create %s %s/%s error: %s"

	//ResourceUpdateError = "update kind namespace/name err: errMsg"
	ResourceUpdateError = "update %s %s/%s error: %s"

	//ResourcePatchError = "patch kind namespace/name err: errMsg"
	ResourcePatchError = "patch %s %s/%s error: %s"

	//ResourceSelectorError = "selector namespace/kind err: errMsg"
	ResourceSelectorError = "selector %s/%s error: %s"

	//ResourceInvalidError = "kind resource is invalid : errMsg"
	ResourceInvalidError = "%s resource is invalid : %s"

	//ShouldNotHappenError = "kind resource is invalid : errMsg"
	ShouldNotHappenError = "should not happen : %s"
)

const ServiceImagePlaceholder = " "

const TarsNodeLabel = "tars.io/node"       // 此标签表示 该节点可以被 tars 使用
const TarsAbilityLabel = "tars.io/ability" // 此标签表示 该节点可以被 tars 当做 App节点池使用

const TemplateLabel = "tars.io/Template"
const TSubTypeLabel = "tars.io/SubType"

const TServerAppLabel = "tars.io/ServerApp"
const TServerNameLabel = "tars.io/ServerName"
const TServerIdLabel = "tars.io/ServerID"
const TConfigNameLabel = "tars.io/ConfigName"
const TConfigVersionLabel = "tars.io/Version"
const TConfigActivatedLabel = "tars.io/Activated"
const TConfigPodSeqLabel = "tars.io/PodSeq"
const TConfigDeletingLabel = "tars.io/Deleting"
const TConfigDeactivateLabel = "tars.io/Deactivate"
const TLocalVolumeLabel = "tars.io/LocalVolume"

const NodeServantName = "nodeobj"
const NodeServantPort = 19385

const TServerKind = "TServer"

const TPodReadinessGate = "tars.io/active"

const K8SHostNameLabel = "kubernetes.io/hostname"

const WebhookCertFile = "/etc/tarscontroller-cert/tls.crt"
const WebhookCertKey = "/etc/tarscontroller-cert/tls.key"

const TControllerServiceAccount = "tars-controller"

const TTreeResourceName = "tars-tree"

const TStorageClassName = "t-storage-class"
const THostBindPlaceholder = "delay-bind"

const HelmFinalizerAccount = "tars-framework-finalizer"
