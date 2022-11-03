package v1beta3

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/integer"
	tarsApisV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsMeta "k8s.tars.io/meta"
)

func buildDaemonset(tserver *tarsApisV1beta3.TServer) *k8sAppsV1.DaemonSet {
	daemonSet := &k8sAppsV1.DaemonSet{
		TypeMeta: k8sMetaV1.TypeMeta{},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserver.Name,
			Namespace: tserver.Namespace,
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, tarsApisV1beta3.SchemeGroupVersion.WithKind(tarsMeta.TServerKind)),
			},
			Labels: map[string]string{
				tarsMeta.TServerAppLabel:  tserver.Spec.App,
				tarsMeta.TServerNameLabel: tserver.Spec.Server,
			},
		},
		Spec: k8sAppsV1.DaemonSetSpec{
			Selector: &k8sMetaV1.LabelSelector{
				MatchLabels: map[string]string{
					tarsMeta.TServerAppLabel:  tserver.Spec.App,
					tarsMeta.TServerNameLabel: tserver.Spec.Server,
				},
			},
			Template:       buildPodTemplate(tserver),
			UpdateStrategy: buildDaemonsetUpdateStrategy(tserver),
		},
	}
	return daemonSet
}

func syncDaemonSet(tserver *tarsApisV1beta3.TServer, daemonSet *k8sAppsV1.DaemonSet) {
	var sst = buildPodTemplate(tserver)
	for _, v := range daemonSet.Spec.Template.Spec.Containers {
		if v.Name != tserver.Name {
			sst.Spec.Containers = append(sst.Spec.Containers, *v.DeepCopy())
		}
	}

	for _, v := range daemonSet.Spec.Template.Spec.InitContainers {
		if v.Name != "tarsnode" {
			sst.Spec.Containers = append(sst.Spec.InitContainers, *v.DeepCopy())
		}
	}
	daemonSet.Spec.Template = sst
}

func buildDaemonsetUpdateStrategy(tserver *tarsApisV1beta3.TServer) k8sAppsV1.DaemonSetUpdateStrategy {
	us := k8sAppsV1.DaemonSetUpdateStrategy{
		Type: k8sAppsV1.DaemonSetUpdateStrategyType(tserver.Spec.K8S.UpdateStrategy.Type),
	}
	if tserver.Spec.K8S.UpdateStrategy.RollingUpdate != nil && tserver.Spec.K8S.UpdateStrategy.RollingUpdate.Partition != nil {
		intValue := intstr.IntOrString{
			Type:   0,
			IntVal: integer.Int32Max(*tserver.Spec.K8S.UpdateStrategy.RollingUpdate.Partition, 1),
		}
		us.RollingUpdate = &k8sAppsV1.RollingUpdateDaemonSet{
			MaxUnavailable: &intValue,
		}
	}
	return us
}
