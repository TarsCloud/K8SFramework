package v1beta3

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	tarsV1beta3 "k8s.tars.io/crd/v1beta3"
)

const DefaultUnlawfulAndOnlyForDebugUserName = "(^_^)"
const DefaultControllerNamespace = "tars-system"
const DefaultMaxRecordLen = 60
const DefaultMaxTConfigHistory = 10
const DefaultMaxTImageRelease = 32
const DefaultMaxImageBuildTime = 480 //second

const DefaultLauncherType = tarsV1beta3.Background
const DefaultImagePullPolicy = k8sCoreV1.PullAlways

var defaultStatefulsetPartition = int32(0)
var DefaultStatefulsetUpdateStrategy = k8sAppsV1.StatefulSetUpdateStrategy{
	Type: k8sAppsV1.RollingUpdateStatefulSetStrategyType,
	RollingUpdate: &k8sAppsV1.RollingUpdateStatefulSetStrategy{
		Partition: &defaultStatefulsetPartition,
	},
}
