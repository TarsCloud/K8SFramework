package meta

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
)

const DefaultControllerNamespace = "tars-system"
const DefaultControllerUsername = "tars-controller"
const DefaultControllerServiceAccount = "system:serviceaccount:" + DefaultControllerNamespace + ":" + "tars-controller"
const DefaultMaxRecordLen = 60
const DefaultMaxTConfigHistory = 10
const DefaultMaxTImageRelease = 32
const DefaultMaxImageBuildTime = 480 //second
const DefaultLauncherType = Background
const DefaultImagePullPolicy = k8sCoreV1.PullAlways

var defaultStatefulsetPartition = int32(0)

var DefaultStatefulsetUpdateStrategy = k8sAppsV1.StatefulSetUpdateStrategy{
	Type: k8sAppsV1.RollingUpdateStatefulSetStrategyType,
	RollingUpdate: &k8sAppsV1.RollingUpdateStatefulSetStrategy{
		Partition: &defaultStatefulsetPartition,
	},
}
