package meta

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
)

const KubernetesSystemAccountPrefix = "system:serviceaccount:kube-system:"

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

const (
	ServiceImagePlaceholder = " "

	TarsNodeLabel          = "tars.io/node"    // 此标签表示 该节点可以被 tars 使用
	TarsAbilityLabelPrefix = "tars.io/ability" // 此标签表示 该节点可以被 tars 当做 App节点池使用

	TemplateLabel = "tars.io/Template"
	ParentLabel   = "tars.io/Parent"
	TSubTypeLabel = "tars.io/SubType"

	TServerAppLabel        = "tars.io/ServerApp"
	TServerNameLabel       = "tars.io/ServerName"
	TServerIdLabel         = "tars.io/ServerID"
	TConfigNameLabel       = "tars.io/ConfigName"
	TConfigVersionLabel    = "tars.io/Version"
	TConfigActivatedLabel  = "tars.io/Activated"
	TConfigPodSeqLabel     = "tars.io/PodSeq"
	TConfigDeletingLabel   = "tars.io/Deleting"
	TConfigDeactivateLabel = "tars.io/Deactivate"
	TLocalVolumeLabel      = "tars.io/LocalVolume"
	TLocalVolumeUIDLabel   = "tars.io/LocalVolumeUID"
	TLocalVolumeModeLabel  = "tars.io/LocalVolumeMode"

	TLocalVolumeGIDLabel = "tars.io/LocalVolumeGID"
	K8SHostNameLabel     = "kubernetes.io/hostname"
)

const (
	TMaxReplicasAnnotation = "tars.io/MaxReplicas"
	TMinReplicasAnnotation = "tars.io/MinReplicas"
)

const TPodReadinessGate = "tars.io/active"
const NodeServantName = "nodeobj"
const NodeServantPort = 19385

const FixedTTreeResourceName = "tars-tree"
const FixedTFrameworkConfigResourceName = "tars-framework"

const TStorageClassName = "tars-storage-class"
const THostBindPlaceholder = "delay-bind"

const (
	TServerKind          = "TServer"
	TImageKind           = "TImage"
	TConfigKind          = "TConfig"
	TAccountKind         = "TAccount"
	TTemplateKind        = "TTemplate"
	TEndpointKind        = "TEndpoint"
	TTreeKind            = "TTree"
	TFrameworkConfigKind = "TFrameworkConfig"
)

const MaxTServerName = 59

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

const DefaultUnlawfulAndOnlyForDebugUserName = "(^_^)"
const DefaultControllerNamespace = "tars-system"
const DefaultMaxRecordLen = 60
const DefaultMaxTConfigHistory = 10
const DefaultMaxTImageRelease = 32
const DefaultMaxImageBuildTime = 480 //second

type LauncherType string

const (
	Foreground LauncherType = "foreground"
	Background LauncherType = "background"
)

const DefaultLauncherType = Background
const DefaultImagePullPolicy = k8sCoreV1.PullAlways

var defaultStatefulsetPartition = int32(0)
var DefaultStatefulsetUpdateStrategy = k8sAppsV1.StatefulSetUpdateStrategy{
	Type: k8sAppsV1.RollingUpdateStatefulSetStrategyType,
	RollingUpdate: &k8sAppsV1.RollingUpdateStatefulSetStrategy{
		Partition: &defaultStatefulsetPartition,
	},
}
