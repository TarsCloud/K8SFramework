package v1beta2

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	tarsCrdV1beta2 "k8s.tars.io/crd/v1beta2"
	tarsMeta "k8s.tars.io/meta"
)

var defaultStatefulsetPartition = int32(0)
var DefaultStatefulsetUpdateStrategy = k8sAppsV1.StatefulSetUpdateStrategy{
	Type: k8sAppsV1.RollingUpdateStatefulSetStrategyType,
	RollingUpdate: &k8sAppsV1.RollingUpdateStatefulSetStrategy{
		Partition: &defaultStatefulsetPartition,
	},
}

type TServerAppend1b11b2 struct {
	UpdateStrategy                     k8sAppsV1.StatefulSetUpdateStrategy `json:"updateStrategy"`
	ImagePullPolicy                    k8sCoreV1.PullPolicy                `json:"imagePullPolicy"`
	LauncherType                       tarsMeta.LauncherType               `json:"launcherType"`
	*tarsCrdV1beta2.TServerReleaseNode `json:",inline"`
}

type TServerDrop1b11b2 struct {
}

type TServerConversion1b11b2 struct {
	Append TServerAppend1b11b2 `json:"append,omitempty"`
	Drop   TServerDrop1b11b2   `json:"drop,omitempty"`
}
