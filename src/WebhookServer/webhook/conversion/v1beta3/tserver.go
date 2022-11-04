package v1beta3

import (
	"fmt"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	tarsAppsV1beta1 "k8s.tars.io/apps/v1beta1"
	tarsAppsV1beta2 "k8s.tars.io/apps/v1beta2"
	tarsAppsV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"unsafe"
)

func conversionTars1b1To1b3(src *tarsAppsV1beta1.TServerTars) *tarsAppsV1beta3.TServerTars {
	if src == nil {
		return nil
	}
	dst := &tarsAppsV1beta3.TServerTars{
		Template:    src.Template,
		Profile:     src.Profile,
		AsyncThread: src.AsyncThread,
		Servants:    []*tarsAppsV1beta3.TServerServant{},
		Ports:       []*tarsAppsV1beta3.TServerPort{},
	}

	for _, p := range src.Servants {
		dst.Servants = append(dst.Servants, (*tarsAppsV1beta3.TServerServant)(p))
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsAppsV1beta3.TServerPort)(p))
	}
	return dst
}

func conversionNormal1b1To1b3(src *tarsAppsV1beta1.TServerNormal) *tarsAppsV1beta3.TServerNormal {
	if src == nil {
		return nil
	}
	dst := &tarsAppsV1beta3.TServerNormal{
		Ports: []*tarsAppsV1beta3.TServerPort{},
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsAppsV1beta3.TServerPort)(p))
	}
	return dst
}

func conversionMount1b1To1b3(src []tarsAppsV1beta1.TK8SMount) []tarsAppsV1beta3.TK8SMount {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []tarsAppsV1beta3.TK8SMount
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func conversionHostPorts1b1To1b3(src []*tarsAppsV1beta1.TK8SHostPort) []*tarsAppsV1beta3.TK8SHostPort {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []*tarsAppsV1beta3.TK8SHostPort
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func CvTServer1b1To1b3(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &tarsAppsV1beta1.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)

		var dst = &tarsAppsV1beta3.TServer{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: tarsMeta.TarsGroupVersionV1B3,
				Kind:       tarsMeta.TServerKind,
			},
			ObjectMeta: src.ObjectMeta,
			Spec: tarsAppsV1beta3.TServerSpec{
				App:       src.Spec.App,
				Server:    src.Spec.Server,
				SubType:   tarsAppsV1beta3.TServerSubType(src.Spec.SubType),
				Important: src.Spec.Important,
				Tars:      conversionTars1b1To1b3(src.Spec.Tars),
				Normal:    conversionNormal1b1To1b3(src.Spec.Normal),
				K8S: tarsAppsV1beta3.TServerK8S{
					ServiceAccount:      src.Spec.K8S.ServiceAccount,
					Args:                []string{},
					Command:             []string{},
					Env:                 src.Spec.K8S.Env,
					EnvFrom:             src.Spec.K8S.EnvFrom,
					HostIPC:             src.Spec.K8S.HostIPC,
					HostNetwork:         src.Spec.K8S.HostNetwork,
					HostPorts:           conversionHostPorts1b1To1b3(src.Spec.K8S.HostPorts),
					Mounts:              conversionMount1b1To1b3(src.Spec.K8S.Mounts),
					DaemonSet:           src.Spec.K8S.DaemonSet,
					NodeSelector:        src.Spec.K8S.NodeSelector,
					AbilityAffinity:     tarsAppsV1beta3.AbilityAffinityType(src.Spec.K8S.AbilityAffinity),
					NotStacked:          src.Spec.K8S.NotStacked,
					PodManagementPolicy: src.Spec.K8S.PodManagementPolicy,
					Replicas:            src.Spec.K8S.Replicas,
					ReadinessGates:      []string{},
					Resources:           src.Spec.K8S.Resources,
					UpdateStrategy:      tarsMeta.DefaultStatefulsetUpdateStrategy,
					ImagePullPolicy:     tarsMeta.DefaultImagePullPolicy,
					LauncherType:        tarsMeta.DefaultLauncherType,
				},
				Release: nil,
			},
			Status: tarsAppsV1beta3.TServerStatus(src.Status),
		}

		if src.Spec.Release != nil {
			dst.Spec.Release = &tarsAppsV1beta3.TServerRelease{
				ID:     src.Spec.Release.ID,
				Image:  src.Spec.Release.Image,
				Secret: src.Spec.Release.Secret,
				Time:   src.Spec.Release.Time,
			}
		}

		if src.Spec.K8S.ReadinessGate != "" {
			dst.Spec.K8S.ReadinessGates = []string{src.Spec.K8S.ReadinessGate}
		}

		for ii := 0; ii < 1; ii++ {
			if src.ObjectMeta.Annotations == nil {
				break
			}
			conversionAnnotation, _ := src.ObjectMeta.Annotations[V1b1V1b3Annotation]
			if conversionAnnotation == "" {
				delete(dst.ObjectMeta.Annotations, V1b1V1b3Annotation)
				break
			}
			var diff = TServerConversion1b11b3{}
			err := json.Unmarshal([]byte(conversionAnnotation), &diff)
			if err != nil {
				utilRuntime.HandleError(fmt.Errorf("read conversion annotation error: %s", err.Error()))
				delete(dst.ObjectMeta.Annotations, V1b1V1b3Annotation)
				break
			}
			dst.Spec.K8S.UpdateStrategy = diff.Append.UpdateStrategy
			dst.Spec.K8S.ImagePullPolicy = diff.Append.ImagePullPolicy
			dst.Spec.K8S.LauncherType = diff.Append.LauncherType
			dst.Spec.K8S.Command = diff.Append.Command
			dst.Spec.K8S.Args = diff.Append.Args
			dst.Spec.K8S.ReadinessGates = append(dst.Spec.K8S.ReadinessGates, diff.Append.ReadinessGates...)
			if dst.Spec.Release != nil {
				dst.Spec.Release.TServerReleaseNode = diff.Append.TServerReleaseNode
			}
		}
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func conversionTars1b3To1b1(src *tarsAppsV1beta3.TServerTars) *tarsAppsV1beta1.TServerTars {
	if src == nil {
		return nil
	}
	dst := &tarsAppsV1beta1.TServerTars{
		Template:    src.Template,
		Profile:     src.Profile,
		AsyncThread: src.AsyncThread,
		Servants:    []*tarsAppsV1beta1.TServerServant{},
		Ports:       []*tarsAppsV1beta1.TServerPort{},
	}

	for _, p := range src.Servants {
		dst.Servants = append(dst.Servants, (*tarsAppsV1beta1.TServerServant)(p))
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsAppsV1beta1.TServerPort)(p))
	}
	return dst
}

func conversionNormal1b3To1b1(src *tarsAppsV1beta3.TServerNormal) *tarsAppsV1beta1.TServerNormal {
	if src == nil {
		return nil
	}
	dst := &tarsAppsV1beta1.TServerNormal{
		Ports: []*tarsAppsV1beta1.TServerPort{},
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsAppsV1beta1.TServerPort)(p))
	}
	return dst
}

func conversionHostPorts1b3To1b1(src []*tarsAppsV1beta3.TK8SHostPort) []*tarsAppsV1beta1.TK8SHostPort {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []*tarsAppsV1beta1.TK8SHostPort
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func conversionMount1b3To1b1(src []tarsAppsV1beta3.TK8SMount) []tarsAppsV1beta1.TK8SMount {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []tarsAppsV1beta1.TK8SMount
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func conversionReadinessGate1b3To1b1(src []string) string {
	if len(src) > 0 {
		return src[0]
	}
	return ""
}

func CvTServer1b3To1b1(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &tarsAppsV1beta3.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)

		var dst = &tarsAppsV1beta1.TServer{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: tarsMeta.TarsGroupVersionV1B1,
				Kind:       tarsMeta.TServerKind,
			},
			ObjectMeta: src.ObjectMeta,
			Spec: tarsAppsV1beta1.TServerSpec{
				App:       src.Spec.App,
				Server:    src.Spec.Server,
				SubType:   tarsAppsV1beta1.TServerSubType(src.Spec.SubType),
				Important: src.Spec.Important,
				Tars:      conversionTars1b3To1b1(src.Spec.Tars),
				Normal:    conversionNormal1b3To1b1(src.Spec.Normal),
				K8S: tarsAppsV1beta1.TServerK8S{
					ServiceAccount:      src.Spec.K8S.ServiceAccount,
					Env:                 src.Spec.K8S.Env,
					EnvFrom:             src.Spec.K8S.EnvFrom,
					HostIPC:             src.Spec.K8S.HostIPC,
					HostNetwork:         src.Spec.K8S.HostNetwork,
					HostPorts:           conversionHostPorts1b3To1b1(src.Spec.K8S.HostPorts),
					Mounts:              conversionMount1b3To1b1(src.Spec.K8S.Mounts),
					DaemonSet:           src.Spec.K8S.DaemonSet,
					NodeSelector:        src.Spec.K8S.NodeSelector,
					AbilityAffinity:     tarsAppsV1beta1.AbilityAffinityType(src.Spec.K8S.AbilityAffinity),
					NotStacked:          src.Spec.K8S.NotStacked,
					PodManagementPolicy: src.Spec.K8S.PodManagementPolicy,
					Replicas:            src.Spec.K8S.Replicas,
					ReadinessGate:       conversionReadinessGate1b3To1b1(src.Spec.K8S.ReadinessGates),
					Resources:           src.Spec.K8S.Resources,
				},
				Release: nil,
			},
			Status: tarsAppsV1beta1.TServerStatus(src.Status),
		}

		if src.Spec.Release != nil {
			dst.Spec.Release = &tarsAppsV1beta1.TServerRelease{
				ID:     src.Spec.Release.ID,
				Image:  src.Spec.Release.Image,
				Secret: src.Spec.Release.Secret,
				Time:   src.Spec.Release.Time,
			}
		}

		diff := TServerConversion1b11b3{
			Append: TServerAppend1b11b3{
				UpdateStrategy:  src.Spec.K8S.UpdateStrategy,
				ImagePullPolicy: src.Spec.K8S.ImagePullPolicy,
				LauncherType:    src.Spec.K8S.LauncherType,
				Command:         src.Spec.K8S.Command,
				Args:            src.Spec.K8S.Args,
			},
		}

		if len(src.Spec.K8S.ReadinessGates) > 1 {
			diff.Append.ReadinessGates = src.Spec.K8S.ReadinessGates[1:]
		}

		if src.Spec.Release != nil {
			diff.Append.TServerReleaseNode = src.Spec.Release.TServerReleaseNode
		}

		bs, _ := json.Marshal(diff)
		if dst.Annotations == nil {
			dst.Annotations = map[string]string{}
		}
		dst.Annotations[V1b1V1b3Annotation] = string(bs)
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func conversionTars1b2To1b3(src *tarsAppsV1beta2.TServerTars) *tarsAppsV1beta3.TServerTars {
	if src == nil {
		return nil
	}
	dst := &tarsAppsV1beta3.TServerTars{
		Template:    src.Template,
		Profile:     src.Profile,
		AsyncThread: src.AsyncThread,
		Servants:    []*tarsAppsV1beta3.TServerServant{},
		Ports:       []*tarsAppsV1beta3.TServerPort{},
	}

	for _, p := range src.Servants {
		dst.Servants = append(dst.Servants, (*tarsAppsV1beta3.TServerServant)(p))
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsAppsV1beta3.TServerPort)(p))
	}
	return dst
}

func conversionNormal1b2To1b3(src *tarsAppsV1beta2.TServerNormal) *tarsAppsV1beta3.TServerNormal {
	if src == nil {
		return nil
	}
	dst := &tarsAppsV1beta3.TServerNormal{
		Ports: []*tarsAppsV1beta3.TServerPort{},
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsAppsV1beta3.TServerPort)(p))
	}
	return dst
}

func conversionMount1b2To1b3(src []tarsAppsV1beta2.TK8SMount) []tarsAppsV1beta3.TK8SMount {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []tarsAppsV1beta3.TK8SMount
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func conversionHostPorts1b2To1b3(src []*tarsAppsV1beta2.TK8SHostPort) []*tarsAppsV1beta3.TK8SHostPort {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []*tarsAppsV1beta3.TK8SHostPort
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func CvTServer1b2To1b3(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &tarsAppsV1beta2.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)

		var dst = &tarsAppsV1beta3.TServer{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: tarsMeta.TarsGroupVersionV1B3,
				Kind:       tarsMeta.TServerKind,
			},
			ObjectMeta: src.ObjectMeta,
			Spec: tarsAppsV1beta3.TServerSpec{
				App:       src.Spec.App,
				Server:    src.Spec.Server,
				SubType:   tarsAppsV1beta3.TServerSubType(src.Spec.SubType),
				Important: src.Spec.Important,
				Tars:      conversionTars1b2To1b3(src.Spec.Tars),
				Normal:    conversionNormal1b2To1b3(src.Spec.Normal),
				K8S: tarsAppsV1beta3.TServerK8S{
					ServiceAccount:      src.Spec.K8S.ServiceAccount,
					Args:                []string{},
					Command:             []string{},
					Env:                 src.Spec.K8S.Env,
					EnvFrom:             src.Spec.K8S.EnvFrom,
					HostIPC:             src.Spec.K8S.HostIPC,
					HostNetwork:         src.Spec.K8S.HostNetwork,
					HostPorts:           conversionHostPorts1b2To1b3(src.Spec.K8S.HostPorts),
					Mounts:              conversionMount1b2To1b3(src.Spec.K8S.Mounts),
					DaemonSet:           src.Spec.K8S.DaemonSet,
					NodeSelector:        src.Spec.K8S.NodeSelector,
					AbilityAffinity:     tarsAppsV1beta3.AbilityAffinityType(src.Spec.K8S.AbilityAffinity),
					NotStacked:          src.Spec.K8S.NotStacked,
					PodManagementPolicy: src.Spec.K8S.PodManagementPolicy,
					Replicas:            src.Spec.K8S.Replicas,
					ReadinessGates:      []string{},
					Resources:           src.Spec.K8S.Resources,
					UpdateStrategy:      src.Spec.K8S.UpdateStrategy,
					ImagePullPolicy:     src.Spec.K8S.ImagePullPolicy,
					LauncherType:        src.Spec.K8S.LauncherType,
				},
				Release: nil,
			},
			Status: tarsAppsV1beta3.TServerStatus(src.Status),
		}

		if src.Spec.Release != nil {
			dst.Spec.Release = (*tarsAppsV1beta3.TServerRelease)(unsafe.Pointer(src.Spec.Release))
		}

		if src.Spec.K8S.ReadinessGate != "" {
			dst.Spec.K8S.ReadinessGates = []string{src.Spec.K8S.ReadinessGate}
		}

		for ii := 0; ii < 1; ii++ {
			if src.ObjectMeta.Annotations == nil {
				break
			}

			conversionAnnotation, _ := src.ObjectMeta.Annotations[V1b2V1b3Annotation]
			if conversionAnnotation == "" {
				delete(dst.ObjectMeta.Annotations, V1b2V1b3Annotation)
				break
			}

			var diff = TServerConversion1b21b3{}
			err := json.Unmarshal([]byte(conversionAnnotation), &diff)
			if err != nil {
				utilRuntime.HandleError(fmt.Errorf("read conversion annotation error: %s", err.Error()))
				delete(dst.ObjectMeta.Annotations, V1b2V1b3Annotation)
				break
			}

			dst.Spec.K8S.Args = diff.Append.Args
			dst.Spec.K8S.Command = diff.Append.Command
			dst.Spec.K8S.ReadinessGates = append(dst.Spec.K8S.ReadinessGates, diff.Append.ReadinessGates...)
		}
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func conversionTars1b3To1b2(src *tarsAppsV1beta3.TServerTars) *tarsAppsV1beta2.TServerTars {
	if src == nil {
		return nil
	}
	dst := &tarsAppsV1beta2.TServerTars{
		Template:    src.Template,
		Profile:     src.Profile,
		AsyncThread: src.AsyncThread,
		Servants:    []*tarsAppsV1beta2.TServerServant{},
		Ports:       []*tarsAppsV1beta2.TServerPort{},
	}

	for _, p := range src.Servants {
		dst.Servants = append(dst.Servants, (*tarsAppsV1beta2.TServerServant)(p))
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsAppsV1beta2.TServerPort)(p))
	}
	return dst
}

func conversionNormal1b3To1b2(src *tarsAppsV1beta3.TServerNormal) *tarsAppsV1beta2.TServerNormal {
	if src == nil {
		return nil
	}
	dst := &tarsAppsV1beta2.TServerNormal{
		Ports: []*tarsAppsV1beta2.TServerPort{},
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, (*tarsAppsV1beta2.TServerPort)(p))
	}
	return dst
}

func conversionHostPorts1b3To1b2(src []*tarsAppsV1beta3.TK8SHostPort) []*tarsAppsV1beta2.TK8SHostPort {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []*tarsAppsV1beta2.TK8SHostPort
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func conversionReadinessGate1b3To1b2(src []string) string {
	if len(src) > 0 {
		return src[0]
	}
	return ""
}

func conversionMount1b3To1b2(src []tarsAppsV1beta3.TK8SMount) []tarsAppsV1beta2.TK8SMount {
	if src == nil {
		return nil
	}
	bs, _ := json.Marshal(src)
	var dst []tarsAppsV1beta2.TK8SMount
	_ = json.Unmarshal(bs, &dst)
	return dst
}

func CvTServer1b3To1b2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &tarsAppsV1beta3.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)

		var dst = &tarsAppsV1beta2.TServer{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: tarsMeta.TarsGroupVersionV1B2,
				Kind:       tarsMeta.TServerKind,
			},
			ObjectMeta: src.ObjectMeta,
			Spec: tarsAppsV1beta2.TServerSpec{
				App:       src.Spec.App,
				Server:    src.Spec.Server,
				SubType:   tarsAppsV1beta2.TServerSubType(src.Spec.SubType),
				Important: src.Spec.Important,
				Tars:      conversionTars1b3To1b2(src.Spec.Tars),
				Normal:    conversionNormal1b3To1b2(src.Spec.Normal),
				K8S: tarsAppsV1beta2.TServerK8S{
					ServiceAccount:      src.Spec.K8S.ServiceAccount,
					Env:                 src.Spec.K8S.Env,
					EnvFrom:             src.Spec.K8S.EnvFrom,
					HostIPC:             src.Spec.K8S.HostIPC,
					HostNetwork:         src.Spec.K8S.HostNetwork,
					HostPorts:           conversionHostPorts1b3To1b2(src.Spec.K8S.HostPorts),
					Mounts:              conversionMount1b3To1b2(src.Spec.K8S.Mounts),
					DaemonSet:           src.Spec.K8S.DaemonSet,
					NodeSelector:        src.Spec.K8S.NodeSelector,
					AbilityAffinity:     tarsAppsV1beta2.AbilityAffinityType(src.Spec.K8S.AbilityAffinity),
					NotStacked:          src.Spec.K8S.NotStacked,
					PodManagementPolicy: src.Spec.K8S.PodManagementPolicy,
					Replicas:            src.Spec.K8S.Replicas,
					ReadinessGate:       conversionReadinessGate1b3To1b2(src.Spec.K8S.ReadinessGates),
					Resources:           src.Spec.K8S.Resources,
					UpdateStrategy:      src.Spec.K8S.UpdateStrategy,
					ImagePullPolicy:     src.Spec.K8S.ImagePullPolicy,
					LauncherType:        src.Spec.K8S.LauncherType,
				},
				Release: nil,
			},
			Status: tarsAppsV1beta2.TServerStatus(src.Status),
		}

		if src.Spec.Release != nil {
			dst.Spec.Release = (*tarsAppsV1beta2.TServerRelease)(unsafe.Pointer(src.Spec.Release))
		}

		diff := TServerConversion1b21b3{
			Append: TServerAppend1b21b3{
				Command: src.Spec.K8S.Command,
				Args:    src.Spec.K8S.Args,
			},
		}

		if len(src.Spec.K8S.ReadinessGates) > 1 {
			diff.Append.ReadinessGates = src.Spec.K8S.ReadinessGates[1:]
		}

		bs, _ := json.Marshal(diff)
		if dst.Annotations == nil {
			dst.Annotations = map[string]string{}
		}
		dst.Annotations[V1b2V1b3Annotation] = string(bs)
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}