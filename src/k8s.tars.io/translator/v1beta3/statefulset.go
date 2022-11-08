package v1beta3

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsApisV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsMeta "k8s.tars.io/meta"
)

func buildTVolumeClaimTemplates(tserver *tarsApisV1beta3.TServer, name string) *k8sCoreV1.PersistentVolumeClaim {
	storageClassName := tarsMeta.TStorageClassName
	volumeMode := k8sCoreV1.PersistentVolumeFilesystem
	quantity, _ := resource.ParseQuantity("1G")
	pvc := &k8sCoreV1.PersistentVolumeClaim{
		TypeMeta: k8sMetaV1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      name,
			Namespace: tserver.Namespace,
			Labels: map[string]string{
				tarsMeta.TServerAppLabel:   tserver.Spec.App,
				tarsMeta.TServerNameLabel:  tserver.Spec.Server,
				tarsMeta.TLocalVolumeLabel: name,
			},
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, tarsApisV1beta3.SchemeGroupVersion.WithKind(tarsMeta.TServerKind)),
			},
		},
		Spec: k8sCoreV1.PersistentVolumeClaimSpec{
			AccessModes: []k8sCoreV1.PersistentVolumeAccessMode{k8sCoreV1.ReadWriteOnce},
			Selector: &k8sMetaV1.LabelSelector{
				MatchLabels: map[string]string{
					tarsMeta.TServerAppLabel:   tserver.Spec.App,
					tarsMeta.TServerNameLabel:  tserver.Spec.Server,
					tarsMeta.TLocalVolumeLabel: name,
				},
			},
			Resources: k8sCoreV1.ResourceRequirements{
				Requests: map[k8sCoreV1.ResourceName]resource.Quantity{
					k8sCoreV1.ResourceStorage: quantity,
				},
			},
			StorageClassName: &storageClassName,
			VolumeMode:       &volumeMode,
		},
	}
	return pvc
}

func buildStatefulsetVolumeClaimTemplates(tserver *tarsApisV1beta3.TServer) []k8sCoreV1.PersistentVolumeClaim {
	var volumeClaimTemplates []k8sCoreV1.PersistentVolumeClaim
	for _, mount := range tserver.Spec.K8S.Mounts {
		if mount.Source.PersistentVolumeClaimTemplate != nil {
			pvc := mount.Source.PersistentVolumeClaimTemplate.DeepCopy()
			pvc.Name = mount.Name
			volumeClaimTemplates = append(volumeClaimTemplates, *pvc)
		}
		if mount.Source.TLocalVolume != nil {
			volumeClaimTemplates = append(volumeClaimTemplates, *buildTVolumeClaimTemplates(tserver, mount.Name))
		}
	}

	if tserver.Spec.K8S.HostIPC || tserver.Spec.K8S.HostNetwork || len(tserver.Spec.K8S.HostPorts) > 0 {
		volumeClaimTemplates = append(volumeClaimTemplates, *buildTVolumeClaimTemplates(tserver, tarsMeta.THostBindPlaceholder))
	}

	return volumeClaimTemplates
}

func buildStatefulsetUpdateStrategy(tserver *tarsApisV1beta3.TServer) k8sAppsV1.StatefulSetUpdateStrategy {
	return tserver.Spec.K8S.UpdateStrategy
}

func buildStatefulset(tserver *tarsApisV1beta3.TServer) *k8sAppsV1.StatefulSet {
	historyLimit := tarsMeta.DefaultWorkloadHistoryLimit
	statefulSet := &k8sAppsV1.StatefulSet{
		TypeMeta: k8sMetaV1.TypeMeta{},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserver.Name,
			Namespace: tserver.Namespace,
			Labels: map[string]string{
				tarsMeta.TServerAppLabel:  tserver.Spec.App,
				tarsMeta.TServerNameLabel: tserver.Spec.Server,
			},
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, tarsApisV1beta3.SchemeGroupVersion.WithKind(tarsMeta.TServerKind)),
			},
		},
		Spec: k8sAppsV1.StatefulSetSpec{
			Replicas: &tserver.Spec.K8S.Replicas,
			Selector: &k8sMetaV1.LabelSelector{
				MatchLabels: map[string]string{
					tarsMeta.TServerAppLabel:  tserver.Spec.App,
					tarsMeta.TServerNameLabel: tserver.Spec.Server,
				},
			},
			Template:             buildPodTemplate(tserver),
			VolumeClaimTemplates: buildStatefulsetVolumeClaimTemplates(tserver),
			ServiceName:          tserver.Name,
			PodManagementPolicy:  tserver.Spec.K8S.PodManagementPolicy,
			UpdateStrategy:       buildStatefulsetUpdateStrategy(tserver),
			RevisionHistoryLimit: &historyLimit,
		},
	}
	return statefulSet
}

func syncStatefulSet(tserver *tarsApisV1beta3.TServer, statefulSet *k8sAppsV1.StatefulSet) {

	statefulSet.Spec.Replicas = &tserver.Spec.K8S.Replicas
	statefulSet.Spec.UpdateStrategy = tserver.Spec.K8S.UpdateStrategy

	var sst = buildPodTemplate(tserver)

	for _, v := range statefulSet.Spec.Template.Spec.Containers {
		if v.Name != tserver.Name {
			sst.Spec.Containers = append(sst.Spec.Containers, *v.DeepCopy())
		}
	}

	for _, v := range statefulSet.Spec.Template.Spec.InitContainers {
		if v.Name != "tarsnode" {
			sst.Spec.Containers = append(sst.Spec.InitContainers, *v.DeepCopy())
		}
	}

	statefulSet.Spec.Template = sst
}
