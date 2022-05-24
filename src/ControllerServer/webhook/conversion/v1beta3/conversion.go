package v1beta3

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
)

type TServerAppend1b21b3 struct {
	Command        []string `json:"command"`
	Args           []string `json:"args"`
	ReadinessGates []string `json:"readinessGates"`
}

type TServerDrop1b21b3 struct {
}

type TServerConversion1b21b3 struct {
	Append TServerAppend1b21b3 `json:"append,omitempty"`
	Drop   TServerDrop1b21b3   `json:"drop,omitempty"`
}

type TServerAppend1b11b3 struct {
	UpdateStrategy                     k8sAppsV1.StatefulSetUpdateStrategy `json:"updateStrategy"`
	ImagePullPolicy                    k8sCoreV1.PullPolicy                `json:"imagePullPolicy"`
	LauncherType                       tarsCrdV1beta3.LauncherType         `json:"launcherType"`
	*tarsCrdV1beta3.TServerReleaseNode `json:",inline"`
	Command                            []string `json:"command"`
	Args                               []string `json:"args"`
	ReadinessGates                     []string `json:"readinessGates"`
}

type TServerDrop1b11b3 struct {
}

type TServerConversion1b11b3 struct {
	Append TServerAppend1b11b3 `json:"append,omitempty"`
	Drop   TServerDrop1b11b3   `json:"drop,omitempty"`
}

type TFCAppend1b21b3 struct {
	Executor tarsCrdV1beta3.TFrameworkImage `json:"executor"`
}

type TFCDrop1b21b3 struct {
}

type TFCConversion1b21b3 struct {
	Append TFCAppend1b21b3 `json:"append,omitempty"`
	Drop   TFCDrop1b21b3   `json:"drop,omitempty"`
}
