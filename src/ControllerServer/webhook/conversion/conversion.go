package conversion

import (
	"encoding/json"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sExtensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	crdV1alpha1 "k8s.tars.io/api/crd/v1alpha1"
	crdV1alpha2 "k8s.tars.io/api/crd/v1alpha2"
	"net/http"
	"tarscontroller/meta"
	"unsafe"
)

type Conversion struct {
}

func New() *Conversion {
	c := &Conversion{}
	return c
}

func (*Conversion) Handle(w http.ResponseWriter, r *http.Request) {
	requestConversionView := &k8sExtensionsV1.ConversionReview{}
	err := json.NewDecoder(r.Body).Decode(&requestConversionView)
	if err != nil {
		return
	}

	v, _ := extractAPIVersion(requestConversionView.Request.Objects[0])

	responseConversionView := &k8sExtensionsV1.ConversionReview{
		TypeMeta: k8sMetaV1.TypeMeta{
			Kind:       requestConversionView.Kind,
			APIVersion: requestConversionView.APIVersion,
		},
		Response: &k8sExtensionsV1.ConversionResponse{
			UID: requestConversionView.Request.UID,
			Result: k8sMetaV1.Status{
				Status: "Success",
			},
			ConvertedObjects: conversionFunctions[v.Kind][v.APIVersion][requestConversionView.Request.DesiredAPIVersion](requestConversionView.Request.Objects),
		},
	}

	responseBytes, _ := json.Marshal(responseConversionView)
	_, _ = w.Write(responseBytes)
}

func extractAPIVersion(in runtime.RawExtension) (*k8sMetaV1.TypeMeta, error) {
	var typeMeta = &k8sMetaV1.TypeMeta{}
	if err := json.Unmarshal(in.Raw, typeMeta); err != nil {
		return nil, err
	}
	return typeMeta, nil
}

func _convertTServerV1alpha2ToV1alpha1(src *crdV1alpha2.TServer) (dst *crdV1alpha1.TServer) {
	dst = &crdV1alpha1.TServer{
		TypeMeta: k8sMetaV1.TypeMeta{
			APIVersion: "k8s.tars.io/v1alpha1",
			Kind:       meta.TServerKind,
		},
		ObjectMeta: src.ObjectMeta,
		Spec: crdV1alpha1.TServerSpec{
			App:       src.Spec.App,
			Server:    src.Spec.Server,
			SubType:   crdV1alpha1.TServerSubType(src.Spec.SubType),
			Important: src.Spec.Important,
			Tars:       (*crdV1alpha1.TServerTars)(unsafe.Pointer(src.Spec.Tars)),
			Normal:    (*crdV1alpha1.TServerNormal)(unsafe.Pointer(src.Spec.Normal)),
			K8S: crdV1alpha1.TServerK8S{
				ServiceAccount:      src.Spec.K8S.ServiceAccount,
				Env:                 src.Spec.K8S.Env,
				EnvFrom:             src.Spec.K8S.EnvFrom,
				HostIPC:             src.Spec.K8S.HostIPC,
				HostNetwork:         src.Spec.K8S.HostNetwork,
				HostPorts:           nil,
				Mounts:              nil,
				NodeSelector:        crdV1alpha1.TK8SNodeSelector{},
				NotStacked:          src.Spec.K8S.NotStacked,
				PodManagementPolicy: src.Spec.K8S.PodManagementPolicy,
				Replicas:            src.Spec.K8S.Replicas,
				ReadinessGate:       src.Spec.K8S.ReadinessGate,
				Resources:           src.Spec.K8S.Resources,
			},
			Release: (*crdV1alpha1.TServerRelease)(src.Spec.Release),
		},
		Status: crdV1alpha1.TServerStatus{
			Replicas:        src.Status.Replicas,
			ReadyReplicas:   src.Status.ReadyReplicas,
			CurrentReplicas: src.Status.CurrentReplicas,
			Selector:        src.Status.Selector,
		},
	}

	if src.Spec.K8S.HostPorts != nil {
		bs, _ := json.Marshal(src.Spec.K8S.HostPorts)
		_ = json.Unmarshal(bs, &dst.Spec.K8S.HostPorts)
	}

	if src.Spec.K8S.DaemonSet {
		dst.Spec.K8S.NodeSelector.DaemonSet = &crdV1alpha1.TK8SNodeSelectorKind{
			Values: []string{},
		}
	} else if len(src.Spec.K8S.NodeSelector) > 0 {
		if len(src.Spec.K8S.NodeSelector) == 1 &&
			src.Spec.K8S.NodeSelector[0].Key == meta.K8SHostNameLabel &&
			src.Spec.K8S.NodeSelector[0].Operator == k8sCoreV1.NodeSelectorOpIn {
			dst.Spec.K8S.NodeSelector.NodeBind = &crdV1alpha1.TK8SNodeSelectorKind{
				Values: src.Spec.K8S.NodeSelector[0].Values,
			}
		} else {
			dst.Spec.K8S.NodeSelector.LabelMatch = src.Spec.K8S.NodeSelector
		}
	} else {
		dst.Spec.K8S.NodeSelector.AbilityPool = &crdV1alpha1.TK8SNodeSelectorKind{
			Values: []string{},
		}
	}

	if src.Spec.K8S.Mounts != nil {
		for _, srcMount := range src.Spec.K8S.Mounts {
			dstMount := &crdV1alpha1.TK8SMount{
				Name:             srcMount.Name,
				ReadOnly:         srcMount.ReadOnly,
				MountPath:        srcMount.MountPath,
				SubPath:          srcMount.SubPath,
				MountPropagation: srcMount.MountPropagation,
				SubPathExpr:      srcMount.SubPathExpr,
				Source: crdV1alpha1.TK8SMountSource{
					HostPath:                      srcMount.Source.HostPath,
					EmptyDir:                      srcMount.Source.EmptyDir,
					Secret:                        srcMount.Source.Secret,
					PersistentVolumeClaim:         srcMount.Source.PersistentVolumeClaim,
					DownwardAPI:                   srcMount.Source.DownwardAPI,
					ConfigMap:                     srcMount.Source.ConfigMap,
					PersistentVolumeClaimTemplate: srcMount.Source.PersistentVolumeClaimTemplate,
					//TLocalVolume:                  nil,
				},
			}
			if srcMount.Source.TLocalVolume != nil {
				dstMount.Source.PersistentVolumeClaimTemplate = meta.BuildTVolumeClainTemplates(src, srcMount.Name)
				dstMount.Source.PersistentVolumeClaimTemplate.Annotations = map[string]string{
					"tars.io/ConversionFromTLV": "",
					meta.TLocalVolumeUIDLabel:  srcMount.Source.TLocalVolume.UID,
					meta.TLocalVolumeGIDLabel:  srcMount.Source.TLocalVolume.GID,
					meta.TLocalVolumeModeLabel: srcMount.Source.TLocalVolume.Mode,
				}
			}
			dst.Spec.K8S.Mounts = append(dst.Spec.K8S.Mounts, *dstMount)
		}
	}
	return dst
}

func _convertTServerV1alpha1ToV1alpha2(src *crdV1alpha1.TServer) (dst *crdV1alpha2.TServer) {
	dst = &crdV1alpha2.TServer{
		TypeMeta: k8sMetaV1.TypeMeta{
			APIVersion: "k8s.tars.io/v1alpha2",
			Kind:       meta.TServerKind,
		},
		ObjectMeta: src.ObjectMeta,
		Spec: crdV1alpha2.TServerSpec{
			App:       src.Spec.App,
			Server:    src.Spec.Server,
			SubType:   crdV1alpha2.TServerSubType(src.Spec.SubType),
			Important: src.Spec.Important,
			Tars:       (*crdV1alpha2.TServerTars)(unsafe.Pointer(src.Spec.Tars)),
			Normal:    (*crdV1alpha2.TServerNormal)(unsafe.Pointer(src.Spec.Normal)),
			K8S: crdV1alpha2.TServerK8S{
				ServiceAccount:      src.Spec.K8S.ServiceAccount,
				Env:                 src.Spec.K8S.Env,
				EnvFrom:             src.Spec.K8S.EnvFrom,
				HostIPC:             src.Spec.K8S.HostIPC,
				HostNetwork:         src.Spec.K8S.HostNetwork,
				HostPorts:           nil,
				Mounts:              nil,
				DaemonSet:           false,
				NodeSelector:        []k8sCoreV1.NodeSelectorRequirement{},
				AbilityAffinity:     crdV1alpha2.AppOrServerPreferred,
				NotStacked:          src.Spec.K8S.NotStacked,
				PodManagementPolicy: src.Spec.K8S.PodManagementPolicy,
				Replicas:            src.Spec.K8S.Replicas,
				ReadinessGate:       src.Spec.K8S.ReadinessGate,
				Resources:           src.Spec.K8S.Resources,
			},
			Release: (*crdV1alpha2.TServerRelease)(src.Spec.Release),
		},
		Status: crdV1alpha2.TServerStatus{
			Replicas:        src.Status.Replicas,
			ReadyReplicas:   src.Status.ReadyReplicas,
			CurrentReplicas: src.Status.CurrentReplicas,
			Selector:        src.Status.Selector,
		},
	}

	if src.Spec.K8S.HostPorts != nil {
		bs, _ := json.Marshal(src.Spec.K8S.HostPorts)
		_ = json.Unmarshal(bs, &dst.Spec.K8S.HostPorts)
	}

	if src.Spec.K8S.NodeSelector.DaemonSet != nil {
		dst.Spec.K8S.DaemonSet = true
	} else if src.Spec.K8S.NodeSelector.NodeBind != nil {
		dst.Spec.K8S.NodeSelector = []k8sCoreV1.NodeSelectorRequirement{
			{
				Key:      meta.K8SHostNameLabel,
				Operator: k8sCoreV1.NodeSelectorOpIn,
				Values:   src.Spec.K8S.NodeSelector.NodeBind.Values,
			},
		}
	} else if src.Spec.K8S.NodeSelector.LabelMatch != nil {
		dst.Spec.K8S.NodeSelector = src.Spec.K8S.NodeSelector.LabelMatch
	}

	if src.Spec.K8S.Mounts != nil {
		for _, mount := range src.Spec.K8S.Mounts {
			newMount := &crdV1alpha2.TK8SMount{
				Name:             mount.Name,
				ReadOnly:         mount.ReadOnly,
				MountPath:        mount.MountPath,
				SubPath:          mount.SubPath,
				MountPropagation: mount.MountPropagation,
				SubPathExpr:      mount.SubPathExpr,
				Source: crdV1alpha2.TK8SMountSource{
					HostPath:              mount.Source.HostPath,
					EmptyDir:              mount.Source.EmptyDir,
					Secret:                mount.Source.Secret,
					PersistentVolumeClaim: mount.Source.PersistentVolumeClaim,
					DownwardAPI:           mount.Source.DownwardAPI,
					ConfigMap:             mount.Source.ConfigMap,
					//PersistentVolumeClaimTemplate: nil,
					//TLocalVolume:                  nil,
				},
			}
			if mount.Source.PersistentVolumeClaimTemplate != nil {
				if mount.Source.PersistentVolumeClaimTemplate.Annotations == nil {
					newMount.Source.PersistentVolumeClaimTemplate = mount.Source.PersistentVolumeClaimTemplate
				} else {
					_, ok := mount.Source.PersistentVolumeClaimTemplate.Annotations["tars.io/ConversionFromTLV"]
					if !ok {
						newMount.Source.PersistentVolumeClaimTemplate = mount.Source.PersistentVolumeClaimTemplate
					} else {
						uid, _ := mount.Source.PersistentVolumeClaimTemplate.Annotations[meta.TLocalVolumeUIDLabel]
						gid, _ := mount.Source.PersistentVolumeClaimTemplate.Annotations[meta.TLocalVolumeGIDLabel]
						mode, _ := mount.Source.PersistentVolumeClaimTemplate.Annotations[meta.TLocalVolumeModeLabel]
						newMount.Source.TLocalVolume = &crdV1alpha2.TLocalVolume{
							UID:  uid,
							GID:  gid,
							Mode: mode,
						}
					}
				}
			}
			dst.Spec.K8S.Mounts = append(dst.Spec.K8S.Mounts, *newMount)
		}
	}
	return dst
}

func convertTServerV1alpha2ToV1alpha1(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i, _ := range s {
		var src = &crdV1alpha2.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)
		dst := _convertTServerV1alpha2ToV1alpha1(src)
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func convertTServerV1alpha1ToV1alpha2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i, _ := range s {
		var src = &crdV1alpha1.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)
		dst := _convertTServerV1alpha1ToV1alpha2(src)
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func convertTDeployV1alpha2ToV1alpha1(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))

	for i, _ := range s {
		var src = &crdV1alpha2.TDeploy{}
		_ = json.Unmarshal(s[i].Raw, src)

		fakeTserver := &crdV1alpha2.TServer{
			Spec: src.Apply,
		}

		var dst = &crdV1alpha1.TDeploy{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: "k8s.tars.io/v1alpha1",
				Kind:       "TDeploy",
			},
			ObjectMeta: src.ObjectMeta,
			Apply:      _convertTServerV1alpha2ToV1alpha1(fakeTserver).Spec,
			Approve:    (*crdV1alpha1.TDeployApprove)(src.Approve),
			Deployed:   src.Deployed,
		}
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func convertTDeployV1alpha1ToV1alpha2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i, _ := range s {
		var src = &crdV1alpha1.TDeploy{}
		_ = json.Unmarshal(s[i].Raw, src)

		fakeTserver := &crdV1alpha1.TServer{
			Spec: src.Apply,
		}

		var dst = &crdV1alpha2.TDeploy{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: "k8s.tars.io/v1alpha2",
				Kind:       "TDeploy",
			},
			ObjectMeta: src.ObjectMeta,
			Apply:      _convertTServerV1alpha1ToV1alpha2(fakeTserver).Spec,
			Approve:    (*crdV1alpha2.TDeployApprove)(src.Approve),
			Deployed:   src.Deployed,
		}
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

// map[Kind]map[FromGV]map[ToGV]func([]runtime.RawExtension) []runtime.RawExtension
var conversionFunctions map[string]map[string]map[string]func([]runtime.RawExtension) []runtime.RawExtension

func init() {
	conversionFunctions = map[string]map[string]map[string]func([]runtime.RawExtension) []runtime.RawExtension{
		"TServer": {
			"k8s.tars.io/v1alpha2": {"k8s.tars.io/v1alpha1": convertTServerV1alpha2ToV1alpha1},
			"k8s.tars.io/v1alpha1": {"k8s.tars.io/v1alpha2": convertTServerV1alpha1ToV1alpha2},
		},
		"TDeploy": {
			"k8s.tars.io/v1alpha2": {"k8s.tars.io/v1alpha1": convertTDeployV1alpha2ToV1alpha1},
			"k8s.tars.io/v1alpha1": {"k8s.tars.io/v1alpha2": convertTDeployV1alpha1ToV1alpha2},
		},
	}
}
