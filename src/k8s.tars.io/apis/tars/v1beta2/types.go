/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, ID 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsMeta "k8s.tars.io/meta"
)

type TServerServant struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	Thread     int32  `json:"thread"`
	Connection int32  `json:"connection"`
	Capacity   int32  `json:"capacity"`
	Timeout    int32  `json:"timeout"`
	IsTars     bool   `json:"isTars"`
	IsTcp      bool   `json:"isTcp"`
}

type TLocalVolume struct {
	UID  string `json:"uid,omitempty"`
	GID  string `json:"gid,omitempty"`
	Mode string `json:"mode,omitempty"`
}

type TK8SMountSource struct {
	// HostPath represents a pre-existing file or directory on the host
	// machine that is directly exposed to the container. This is generally
	// used for system agents or other privileged things that are allowed
	// to see the host machine. Most containers will NOT need this.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath
	// ---
	// TODO(jonesdl) We need to restrict who can use host directory mounts and who can/can not
	// mount host directories as read/write.
	// +optional
	HostPath *k8sCoreV1.HostPathVolumeSource `json:"hostPath,omitempty"`
	// EmptyDir represents a temporary directory that shares a pod's lifetime.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir
	// +optional
	EmptyDir *k8sCoreV1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
	// Secret represents a secret that should populate this volume.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#secret
	// +optional
	Secret *k8sCoreV1.SecretVolumeSource `json:"secret,omitempty"`
	// PersistentVolumeClaimVolumeSource represents a reference to a
	// PersistentVolumeClaim in the same namespace.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *k8sCoreV1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// persistentVolumeClaimTemplates is a list of claims that pods are allowed to reference.
	// The StatefulSet controller is responsible for mapping network identities to
	// claims in a way that maintains the identity of a pod. Every claim in
	// this list must have at least one matching (by name) volumeMount in one
	// container in the template. A claim in this list takes precedence over
	// any volumes in the template, with the same name.
	// TODO: Define the behavior if a claim already exists with the same name.
	// +optional
	PersistentVolumeClaimTemplate *k8sCoreV1.PersistentVolumeClaim `json:"persistentVolumeClaimTemplate,omitempty"`
	// TWrapperVolumeClaimTemplates is  wrapper edition volumeClaimTemplates
	// +optional
	TLocalVolume *TLocalVolume `json:"tLocalVolume,omitempty"`
	// DownwardAPI represents downward API about the pod that should populate this volume
	// +optional
	DownwardAPI *k8sCoreV1.DownwardAPIVolumeSource `json:"downwardAPI,omitempty"`
	// ConfigMap represents a configMap that should populate this volume
	// +optional
	ConfigMap *k8sCoreV1.ConfigMapVolumeSource `json:"configMap,omitempty"`
}

type TK8SMount struct {
	Name string `json:"name"`

	Source TK8SMountSource `json:"source"`

	// Mounted read-only if true, read-write otherwise (false or unspecified).
	// Defaults to false.
	// +optional
	ReadOnly bool `json:"readOnly,omitempty"`
	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `json:"mountPath"`
	// Path within the volume from which the container's volume should be mounted.
	// Defaults to "" (volume's root).
	// +optional
	SubPath string `json:"subPath,omitempty"`
	//// mountPropagation determines how mounts are propagated from the host
	//// to container and the other way around.
	//// When not set, MountPropagationNone is used.
	//// This field is beta in 1.10.
	//// +optional
	MountPropagation *k8sCoreV1.MountPropagationMode `json:"mountPropagation,omitempty"`
	// Expanded path within the volume from which the container's volume should be mounted.
	// Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment.
	// Defaults to "" (volume's root).
	// SubPathExpr and SubPath are mutually exclusive.
	// +optional
	SubPathExpr string `json:"subPathExpr,omitempty"`
}

type TServerReleaseNode struct {
	Image  string `json:"nodeImage,omitempty"`
	Secret string `json:"nodeSecret,omitempty"`
}

type TServerRelease struct {
	ID                  string          `json:"id"`
	Image               string          `json:"image"`
	Secret              string          `json:"secret"`
	Time                *k8sMetaV1.Time `json:"time,omitempty"`
	*TServerReleaseNode `json:",inline"`
}

type AbilityAffinityType string

const (
	AppRequired          AbilityAffinityType = "AppRequired"
	ServerRequired       AbilityAffinityType = "ServerRequired"
	AppOrServerPreferred AbilityAffinityType = "AppOrServerPreferred"
	None                 AbilityAffinityType = "None"
)

type TServerK8S struct {
	ServiceAccount string `json:"serviceAccount,omitempty"`

	Env []k8sCoreV1.EnvVar `json:"env,omitempty"`

	EnvFrom []k8sCoreV1.EnvFromSource `json:"envFrom,omitempty"`

	HostIPC bool `json:"hostIPC,omitempty"`

	HostNetwork bool `json:"hostNetwork,omitempty"`

	HostPorts []*TK8SHostPort `json:"hostPorts,omitempty"`

	Mounts []TK8SMount `json:"mounts,omitempty"`

	DaemonSet bool `json:"daemonSet,omitempty"`

	NodeSelector []k8sCoreV1.NodeSelectorRequirement `json:"nodeSelector"`

	AbilityAffinity AbilityAffinityType `json:"abilityAffinity"`

	NotStacked bool `json:"notStacked"`

	PodManagementPolicy k8sAppsV1.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`

	Replicas int32 `json:"replicas"`

	//LivenessProbe  *k8sCoreV1.Probe `json:"livenessProbe,omitempty"`
	//ReadinessProbe *k8sCoreV1.Probe `json:"readinessProbe,omitempty"`

	ReadinessGate string `json:"readinessGate,omitempty"`

	Resources       k8sCoreV1.ResourceRequirements      `json:"resources,omitempty"`
	UpdateStrategy  k8sAppsV1.StatefulSetUpdateStrategy `json:"updateStrategy"`
	ImagePullPolicy k8sCoreV1.PullPolicy                `json:"imagePullPolicy"`
	LauncherType    tarsMeta.LauncherType               `json:"launcherType"`
}

type TK8SHostPort struct {
	NameRef string `json:"nameRef"`
	Port    int32  `json:"port"`
}

type TServerExternalAddress struct {
	IP   string `json:"ip"`
	Port int32  `json:"port"`
}

type TServerExternalUPStream struct {
	Name      string                   `json:"name"`
	IsTcp     bool                     `json:"isTcp"`
	Addresses []TServerExternalAddress `json:"addresses"`
}

type TServerExternal struct {
	Upstreams []TServerExternalUPStream `json:"upstreams"`
}

type TServerTars struct {
	Template    string            `json:"template"`
	Profile     string            `json:"profile"`
	AsyncThread int32             `json:"asyncThread"`
	Servants    []*TServerServant `json:"servants"`
	Ports       []*TServerPort    `json:"ports,omitempty"`
}

type TServerPort struct {
	Name  string `json:"name"`
	Port  int32  `json:"port"`
	IsTcp bool   `json:"isTcp"`
}

type TServerNormal struct {
	Ports []*TServerPort `json:"ports"`
}

type TServerSubType string

const (
	TARS   TServerSubType = "tars"
	Normal TServerSubType = "normal"
)

type TServerSpec struct {
	App       string          `json:"app"`
	Server    string          `json:"server"`
	SubType   TServerSubType  `json:"subType"`
	Important int32           `json:"important"`
	Tars      *TServerTars    `json:"tars,omitempty"`
	Normal    *TServerNormal  `json:"normal,omitempty"`
	K8S       TServerK8S      `json:"k8s"`
	Release   *TServerRelease `json:"release,omitempty"`
}

type TServerStatus struct {
	Replicas        int32  `json:"replicas"`
	ReadyReplicas   int32  `json:"readyReplicas"`
	CurrentReplicas int32  `json:"currentReplicas"`
	Selector        string `json:"selector"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TServer struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	Spec                 TServerSpec   `json:"spec"`
	Status               TServerStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TServerList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`
	Items              []TServer `json:"items"`
}

type TEndpointSpec struct {
	App       string          `json:"app"`
	Server    string          `json:"server"`
	SubType   TServerSubType  `json:"subType"`
	Important int32           `json:"important"`
	Tars      *TServerTars    `json:"tars,omitempty"`
	Normal    *TServerNormal  `json:"normal,omitempty"`
	HostPorts []*TK8SHostPort `json:"hostPorts,omitempty"`
	Release   *TServerRelease `json:"release,omitempty"`
}

type TEndpointPodStatus struct {
	UID               string                      `json:"uid"`
	PID               string                      `json:"pid"`
	Name              string                      `json:"name"`
	PodIP             string                      `json:"podIP"`
	HostIP            string                      `json:"hostIP"`
	StartTime         k8sMetaV1.Time              `json:"startTime,omitempty"`
	ContainerStatuses []k8sCoreV1.ContainerStatus `json:"containerStatuses,omitempty"`
	SettingState      string                      `json:"settingState"`
	PresentState      string                      `json:"presentState"`
	PresentMessage    string                      `json:"presentMessage"`
	ID                string                      `json:"id"`
}

type TEndpointStatus struct {
	PodStatus []*TEndpointPodStatus `json:"pods"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TEndpoint struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	Spec                 TEndpointSpec   `json:"spec"`
	Status               TEndpointStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TEndpointList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`
	Items              []TEndpoint `json:"items"`
}

type TTemplateSpec struct {
	Content string `json:"content"`
	Parent  string `json:"parent"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TTemplate struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	Spec                 TTemplateSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TTemplateList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`

	Items []TTemplate `json:"items"`
}

type TTreeBusiness struct {
	Name         string         `json:"name"`
	Show         string         `json:"show"`
	Weight       int32          `json:"weight"`
	CreatePerson string         `json:"createPerson,omitempty"`
	CreateTime   k8sMetaV1.Time `json:"createTime,omitempty"`
	Mark         string         `json:"mark,omitempty"`
}

type TTreeApp struct {
	Name         string         `json:"name"`
	BusinessRef  string         `json:"businessRef"`
	CreatePerson string         `json:"createPerson,omitempty"`
	CreateTime   k8sMetaV1.Time `json:"createTime,omitempty"`
	Mark         string         `json:"mark,omitempty"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TTree struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	Businesses           []TTreeBusiness `json:"businesses"`
	Apps                 []TTreeApp      `json:"apps"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TTreeList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`

	Items []TTree `json:"items"`
}

type TExitedPod struct {
	UID        string         `json:"uid"`
	Name       string         `json:"name"`
	ID         string         `json:"id"`
	NodeIP     string         `json:"nodeIP"`
	PodIP      string         `json:"podIP"`
	CreateTime k8sMetaV1.Time `json:"createTime"`
	DeleteTime k8sMetaV1.Time `json:"deleteTime"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TExitedRecord struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	App                  string       `json:"app"`
	Server               string       `json:"server"`
	Pods                 []TExitedPod `json:"pods"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TExitedRecordList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`

	Items []TExitedRecord `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TConfig struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	App                  string         `json:"app"`
	Server               string         `json:"server"`
	PodSeq               string         `json:"podSeq"`
	ConfigName           string         `json:"configName"`
	Version              string         `json:"version"`
	ConfigContent        string         `json:"configContent"`
	UpdateTime           k8sMetaV1.Time `json:"updateTime"`
	UpdatePerson         string         `json:"updatePerson"`
	UpdateReason         string         `json:"updateReason"`
	Activated            bool           `json:"activated"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TConfigList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`
	Items              []TConfig `json:"items"`
}

type TAccountAuthenticationToken struct {
	Name           string         `json:"name"`
	Content        string         `json:"content"`
	UpdateTime     k8sMetaV1.Time `json:"updateTime,omitempty"`
	ExpirationTime k8sMetaV1.Time `json:"expirationTime"`
	Valid          bool           `json:"valid,omitempty"`
}

type TAccountAuthentication struct {
	Password       *string                        `json:"password,omitempty"`
	BCryptPassword *string                        `json:"bcryptPassword,omitempty"`
	Tokens         []*TAccountAuthenticationToken `json:"tokens"`
	Activated      bool                           `json:"activated"`
}

type TAccountAuthorization struct {
	Flag       string         `json:"flag"`
	Role       string         `json:"role"`
	UpdateTime k8sMetaV1.Time `json:"updateTime"`
}

type TAccountRoleElem struct {
	App     string   `json:"app"`
	Servers []string `json:"servers"`
}

type TAccountSpec struct {
	Username       string                   `json:"username"`
	Extra          []string                 `json:"extra,omitempty"`
	Authentication TAccountAuthentication   `json:"authentication"`
	Authorization  []*TAccountAuthorization `json:"authorization"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TAccount struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	Spec                 TAccountSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TAccountList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`
	Items              []TAccount `json:"items"`
}

type TImageRelease struct {
	ID           string         `json:"id"`
	Image        string         `json:"image"`
	Secret       string         `json:"secret"`
	CreatePerson *string        `json:"createPerson,omitempty"`
	CreateTime   k8sMetaV1.Time `json:"createTime,omitempty"`
	Mark         *string        `json:"mark,omitempty"`
}

type TImageBuildState struct {
	ID              string         `json:"id"`
	BaseImage       string         `json:"baseImage"`
	BaseImageSecret string         `json:"baseImageSecret"`
	Image           string         `json:"image"`
	Secret          string         `json:"secret"`
	ServerType      string         `json:"serverType"`
	CreatePerson    string         `json:"createPerson"`
	CreateTime      k8sMetaV1.Time `json:"createTime,omitempty"`
	Mark            string         `json:"mark"`
	Phase           string         `json:"phase"`
	Message         string         `json:"message"`
	Handler         string         `json:"handler"`
}

type TImageBuild struct {
	Last    *TImageBuildState `json:"last,omitempty"`
	Running *TImageBuildState `json:"running,omitempty"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TImage struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	ImageType            string           `json:"imageType"`
	SupportedType        []string         `json:"supportedType,omitempty"`
	Releases             []*TImageRelease `json:"releases"`
	Default              *string          `json:"default,omitempty"`
	Build                *TImageBuild     `json:"build,omitempty"`
	Mark                 string           `json:"mark"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TImageList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`
	Items              []TImage `json:"items"`
}

type TFrameworkImageBuild struct {
	MaxBuildTime int    `json:"maxBuildTime"`
	TagFormat    string `json:"tagFormat"`
}

type TFrameworkImageRegistry struct {
	Registry string `json:"registry"`
	Secret   string `json:"secret"`
}

type TFrameworkRecordLimit struct {
	TExitedPod     int `json:"texitedPod"`
	TConfigHistory int `json:"tconfigHistory"`
	TImageRelease  int `json:"timageRelease"`
}

type TFrameworkNodeImage struct {
	Image  string `json:"image,omitempty"`
	Secret string `json:"secret,omitempty"`
}

type TFrameworkTarsEndpoint struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Timeout int    `json:"timeout,omitempty"`
	IsTcp   *bool  `json:"isTcp,omitempty"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TFrameworkConfig struct {
	k8sMetaV1.TypeMeta   `json:",inline"`
	k8sMetaV1.ObjectMeta `json:"metadata,omitempty"`
	ImageBuild           TFrameworkImageBuild                 `json:"imageBuild"`
	ImageRegistry        TFrameworkImageRegistry              `json:"imageRegistry"`
	RecordLimit          TFrameworkRecordLimit                `json:"recordLimit"`
	NodeImage            TFrameworkNodeImage                  `json:"nodeImage"`
	UPChain              map[string][]*TFrameworkTarsEndpoint `json:"upChain"`
	Expand               map[string]string                    `json:"expand"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TFrameworkConfigList struct {
	k8sMetaV1.TypeMeta `json:",inline"`
	k8sMetaV1.ListMeta `json:"metadata"`
	Items              []TFrameworkConfig `json:"items"`
}
