package v1beta2

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"
	tarsV1beta1 "k8s.tars.io/apis/tars/v1beta1"
	tarsV1beta2 "k8s.tars.io/apis/tars/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"tarswebhook/webhook/conversion"
)

func conversionTars1b1To1b2(src *tarsV1beta1.TServerTars) *tarsV1beta2.TServerTars {
	if src == nil {
		return nil
	}
	dst := &tarsV1beta2.TServerTars{
		Template:    src.Template,
		Profile:     src.Profile,
		AsyncThread: src.AsyncThread,
		Servants:    []*tarsV1beta2.TServerServant{},
		Ports:       []*tarsV1beta2.TServerPort{},
	}

	for _, p := range src.Servants {
		dst.Servants = append(dst.Servants, (*tarsV1beta2.TServerServant)(p))
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsV1beta2.TServerPort)(p))
	}
	return dst
}

func conversionNormal1b1To1b2(src *tarsV1beta1.TServerNormal) *tarsV1beta2.TServerNormal {
	if src == nil {
		return nil
	}
	dst := &tarsV1beta2.TServerNormal{
		Ports: []*tarsV1beta2.TServerPort{},
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsV1beta2.TServerPort)(p))
	}
	return dst
}

func conversionMount1b1To1b2(src []tarsV1beta1.TK8SMount) []tarsV1beta2.TK8SMount {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []tarsV1beta2.TK8SMount
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func conversionHostPorts1b1To1b2(src []*tarsV1beta1.TK8SHostPort) []*tarsV1beta2.TK8SHostPort {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []*tarsV1beta2.TK8SHostPort
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func CvTServer1b1To1b2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &tarsV1beta1.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)

		dst := &tarsV1beta2.TServer{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: tarsMeta.TarsGroupVersionV1B2,
				Kind:       tarsMeta.TServerKind,
			},
			ObjectMeta: src.ObjectMeta,
			Spec: tarsV1beta2.TServerSpec{
				App:       src.Spec.App,
				Server:    src.Spec.Server,
				SubType:   tarsV1beta2.TServerSubType(src.Spec.SubType),
				Important: src.Spec.Important,
				Tars:      conversionTars1b1To1b2(src.Spec.Tars),
				Normal:    conversionNormal1b1To1b2(src.Spec.Normal),
				K8S: tarsV1beta2.TServerK8S{
					ServiceAccount:      src.Spec.K8S.ServiceAccount,
					Env:                 src.Spec.K8S.Env,
					EnvFrom:             src.Spec.K8S.EnvFrom,
					HostIPC:             src.Spec.K8S.HostIPC,
					HostNetwork:         src.Spec.K8S.HostNetwork,
					HostPorts:           conversionHostPorts1b1To1b2(src.Spec.K8S.HostPorts),
					Mounts:              conversionMount1b1To1b2(src.Spec.K8S.Mounts),
					DaemonSet:           src.Spec.K8S.DaemonSet,
					NodeSelector:        src.Spec.K8S.NodeSelector,
					AbilityAffinity:     tarsV1beta2.AbilityAffinityType(src.Spec.K8S.AbilityAffinity),
					NotStacked:          src.Spec.K8S.NotStacked,
					PodManagementPolicy: src.Spec.K8S.PodManagementPolicy,
					Replicas:            src.Spec.K8S.Replicas,
					ReadinessGate:       src.Spec.K8S.ReadinessGate,
					Resources:           src.Spec.K8S.Resources,
					UpdateStrategy:      k8sAppsV1.StatefulSetUpdateStrategy{},
					ImagePullPolicy:     tarsMeta.DefaultImagePullPolicy,
					LauncherType:        tarsMeta.DefaultLauncherType,
				},
				Release: nil,
			},
			Status: tarsV1beta2.TServerStatus(src.Status),
		}

		if src.Spec.Release != nil {
			dst.Spec.Release = &tarsV1beta2.TServerRelease{
				ID:                 src.Spec.Release.ID,
				Image:              src.Spec.Release.Image,
				Secret:             src.Spec.Release.Secret,
				Time:               src.Spec.Release.Time,
				TServerReleaseNode: nil,
			}
		}

		var conversionAnnotation string
		if src.ObjectMeta.Annotations != nil {
			conversionAnnotation, _ = src.ObjectMeta.Annotations[V1b1AndV1b2Annotation]
		}

		for ii := 0; ii < 1; ii++ {
			if conversionAnnotation != "" {
				var diff = TServerConversion1b11b2{}
				err := json.Unmarshal([]byte(conversionAnnotation), &diff)
				if err == nil {
					dst.Spec.K8S.UpdateStrategy = diff.Append.UpdateStrategy
					dst.Spec.K8S.ImagePullPolicy = diff.Append.ImagePullPolicy
					dst.Spec.K8S.LauncherType = diff.Append.LauncherType
					if dst.Spec.Release != nil && diff.Append.TServerReleaseNode != nil {
						dst.Spec.Release.TServerReleaseNode = diff.Append.TServerReleaseNode
					}
					break
				}
				klog.Errorf("read conversion annotation error: %s", err.Error())
			}

			dst.Spec.K8S.UpdateStrategy = DefaultStatefulsetUpdateStrategy
			dst.Spec.K8S.ImagePullPolicy = tarsMeta.DefaultImagePullPolicy
			dst.Spec.K8S.LauncherType = tarsMeta.DefaultLauncherType
			if dst.Spec.Tars != nil && dst.Spec.Release != nil {
				image, secret := tarsRuntime.TFCConfig.GetDefaultNodeImage(src.Namespace)
				dst.Spec.Release.TServerReleaseNode = &tarsV1beta2.TServerReleaseNode{
					Image:  image,
					Secret: secret,
				}
			}
		}

		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func conversionTars1b2To1b1(src *tarsV1beta2.TServerTars) *tarsV1beta1.TServerTars {
	if src == nil {
		return nil
	}
	dst := &tarsV1beta1.TServerTars{
		Template:    src.Template,
		Profile:     src.Profile,
		AsyncThread: src.AsyncThread,
		Servants:    []*tarsV1beta1.TServerServant{},
		Ports:       []*tarsV1beta1.TServerPort{},
	}

	for _, p := range src.Servants {
		dst.Servants = append(dst.Servants, (*tarsV1beta1.TServerServant)(p))
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsV1beta1.TServerPort)(p))
	}
	return dst
}

func conversionNormal1b2To1b1(src *tarsV1beta2.TServerNormal) *tarsV1beta1.TServerNormal {
	if src == nil {
		return nil
	}
	dst := &tarsV1beta1.TServerNormal{
		Ports: []*tarsV1beta1.TServerPort{},
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsV1beta1.TServerPort)(p))
	}
	return dst
}

func conversionMount1b2To1b1(src []tarsV1beta2.TK8SMount) []tarsV1beta1.TK8SMount {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []tarsV1beta1.TK8SMount
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func conversionHostPorts1b2To1b1(src []*tarsV1beta2.TK8SHostPort) []*tarsV1beta1.TK8SHostPort {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []*tarsV1beta1.TK8SHostPort
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func CvTServer1b2To1b1(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &tarsV1beta2.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)

		dst := &tarsV1beta1.TServer{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: tarsMeta.TarsGroupVersionV1B1,
				Kind:       tarsMeta.TServerKind,
			},
			ObjectMeta: src.ObjectMeta,
			Spec: tarsV1beta1.TServerSpec{
				App:       src.Spec.App,
				Server:    src.Spec.Server,
				SubType:   tarsV1beta1.TServerSubType(src.Spec.SubType),
				Important: src.Spec.Important,
				Tars:      conversionTars1b2To1b1(src.Spec.Tars),
				Normal:    conversionNormal1b2To1b1(src.Spec.Normal),
				K8S: tarsV1beta1.TServerK8S{
					ServiceAccount:      src.Spec.K8S.ServiceAccount,
					Env:                 src.Spec.K8S.Env,
					EnvFrom:             src.Spec.K8S.EnvFrom,
					HostIPC:             src.Spec.K8S.HostIPC,
					HostNetwork:         src.Spec.K8S.HostNetwork,
					HostPorts:           conversionHostPorts1b2To1b1(src.Spec.K8S.HostPorts),
					Mounts:              conversionMount1b2To1b1(src.Spec.K8S.Mounts),
					DaemonSet:           src.Spec.K8S.DaemonSet,
					NodeSelector:        src.Spec.K8S.NodeSelector,
					AbilityAffinity:     tarsV1beta1.AbilityAffinityType(src.Spec.K8S.AbilityAffinity),
					NotStacked:          src.Spec.K8S.NotStacked,
					PodManagementPolicy: src.Spec.K8S.PodManagementPolicy,
					Replicas:            src.Spec.K8S.Replicas,
					ReadinessGate:       src.Spec.K8S.ReadinessGate,
					Resources:           src.Spec.K8S.Resources,
				},
				Release: nil,
			},
			Status: tarsV1beta1.TServerStatus(src.Status),
		}

		if src.Spec.Release != nil {
			dst.Spec.Release = &tarsV1beta1.TServerRelease{
				ID:     src.Spec.Release.ID,
				Image:  src.Spec.Release.Image,
				Secret: src.Spec.Release.Secret,
				Time:   src.Spec.Release.Time,
			}
		}

		diff := TServerConversion1b11b2{
			Append: TServerAppend1b11b2{
				UpdateStrategy:  src.Spec.K8S.UpdateStrategy,
				ImagePullPolicy: src.Spec.K8S.ImagePullPolicy,
				LauncherType:    src.Spec.K8S.LauncherType,
			},
		}

		if src.Spec.Release != nil {
			diff.Append.TServerReleaseNode = src.Spec.Release.TServerReleaseNode
		}

		bs, _ := json.Marshal(diff)
		if dst.Annotations == nil {
			dst.Annotations = map[string]string{}
		}
		dst.Annotations[V1b1AndV1b2Annotation] = string(bs)
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func init() {
	conversion.Registry(tarsMeta.TServerKind, tarsV1beta2.SchemeGroupVersion, tarsV1beta1.SchemeGroupVersion, CvTServer1b2To1b1)
	conversion.Registry(tarsMeta.TServerKind, tarsV1beta1.SchemeGroupVersion, tarsV1beta2.SchemeGroupVersion, CvTServer1b1To1b2)
}
