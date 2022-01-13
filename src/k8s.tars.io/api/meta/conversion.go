package meta

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
)

type TServerAppend1a21b2 struct {
	UpdateStrategy                 k8sAppsV1.StatefulSetUpdateStrategy `json:"updateStrategy"`
	ImagePullPolicy                k8sCoreV1.PullPolicy                `json:"imagePullPolicy"`
	LauncherType                   crdV1beta2.LauncherType             `json:"launcherType"`
	*crdV1beta2.TServerReleaseNode `json:",inline"`
}

type TServerDrop1a21b2 struct {
}

type TServerConversion1a21b2 struct {
	Append TServerAppend1a21b2 `json:"append"`
	Drop   TServerDrop1a21b2   `json:"drop"`
}
