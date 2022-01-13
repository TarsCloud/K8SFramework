package v1beta2

import (
	"fmt"
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/integer"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	"strings"
)

func buildPodVolumes(tserver *crdV1beta2.TServer) []k8sCoreV1.Volume {
	mounts := tserver.Spec.K8S.Mounts
	var volumes []k8sCoreV1.Volume

	for _, mount := range mounts {
		if mount.Source.PersistentVolumeClaimTemplate != nil || mount.Source.TLocalVolume != nil {
			continue
		}
		volume := k8sCoreV1.Volume{
			Name: mount.Name,
			VolumeSource: k8sCoreV1.VolumeSource{
				HostPath:              mount.Source.HostPath,
				EmptyDir:              mount.Source.EmptyDir,
				Secret:                mount.Source.Secret,
				PersistentVolumeClaim: mount.Source.PersistentVolumeClaim,
				DownwardAPI:           mount.Source.DownwardAPI,
				ConfigMap:             mount.Source.ConfigMap,
			},
		}
		volumes = append(volumes, volume)
	}

	volumes = append(volumes, k8sCoreV1.Volume{
		Name: "host-timezone",
		VolumeSource: k8sCoreV1.VolumeSource{
			HostPath: &k8sCoreV1.HostPathVolumeSource{
				Path: "/etc/localtime",
			},
		}})

	if tserver.Spec.SubType == crdV1beta2.TARS {
		volumes = append(volumes, k8sCoreV1.Volume{
			Name: "tarsnode-work-dir",
			VolumeSource: k8sCoreV1.VolumeSource{
				EmptyDir: &k8sCoreV1.EmptyDirVolumeSource{},
			}})
	}
	return volumes
}

func buildContainerVolumeMounts(tserver *crdV1beta2.TServer) []k8sCoreV1.VolumeMount {
	mounts := tserver.Spec.K8S.Mounts
	var volumeMounts []k8sCoreV1.VolumeMount

	for _, mount := range mounts {
		if tserver.Spec.K8S.DaemonSet {
			if mount.Source.TLocalVolume != nil || mount.Source.PersistentVolumeClaimTemplate != nil {
				continue
			}
		}
		volumeMount := k8sCoreV1.VolumeMount{
			Name:             mount.Name,
			ReadOnly:         mount.ReadOnly,
			MountPath:        mount.MountPath,
			SubPath:          mount.SubPath,
			MountPropagation: mount.MountPropagation,
			SubPathExpr:      mount.SubPathExpr,
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}

	volumeMounts = append(volumeMounts, k8sCoreV1.VolumeMount{
		Name:      "host-timezone",
		MountPath: "/etc/localtime",
	})

	if tserver.Spec.SubType == crdV1beta2.TARS {
		volumeMounts = append(volumeMounts, k8sCoreV1.VolumeMount{
			Name:      "tarsnode-work-dir",
			MountPath: "/usr/local/app/tars/tarsnode",
		})
	}
	return volumeMounts
}

func buildPodInitContainers(tserver *crdV1beta2.TServer) []k8sCoreV1.Container {
	if tserver.Spec.SubType != crdV1beta2.TARS {
		return nil
	}

	var image string
	if tserver.Spec.Release != nil && tserver.Spec.Release.TServerReleaseNode != nil {
		image = tserver.Spec.Release.TServerReleaseNode.Image
	}

	if image == "" {
		utilRuntime.HandleError(fmt.Errorf(crdMeta.ShouldNotHappenError, "no node image set"))
		image = crdMeta.ServiceImagePlaceholder
	}

	containers := []k8sCoreV1.Container{
		{
			Name: "tarsnode",
			Env: []k8sCoreV1.EnvVar{
				{
					Name: "Namespace",
					ValueFrom: &k8sCoreV1.EnvVarSource{
						FieldRef: &k8sCoreV1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				},
				{
					Name: "PodName",
					ValueFrom: &k8sCoreV1.EnvVarSource{
						FieldRef: &k8sCoreV1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
				{
					Name: "PodIP",
					ValueFrom: &k8sCoreV1.EnvVarSource{
						FieldRef: &k8sCoreV1.ObjectFieldSelector{
							FieldPath: "status.podIP",
						},
					},
				},
				{
					Name:  "ServerApp",
					Value: tserver.Spec.App,
				},
				{
					Name:  "ServerName",
					Value: tserver.Spec.Server,
				},
			},
			Resources: k8sCoreV1.ResourceRequirements{},
			VolumeMounts: []k8sCoreV1.VolumeMount{
				{
					Name:      "tarsnode-work-dir",
					MountPath: "/usr/local/app/tars/tarsnode",
				},
			},
			Image:           image,
			ImagePullPolicy: k8sCoreV1.PullAlways,
		},
	}

	if tserver.Spec.K8S.LauncherType != crdV1beta2.Background {
		containers[0].Env = append(containers[0].Env,
			k8sCoreV1.EnvVar{
				Name:  "LauncherType",
				Value: string(tserver.Spec.K8S.LauncherType),
			})
	}

	return containers
}

func buildPodImagePullSecrets(tserver *crdV1beta2.TServer) []k8sCoreV1.LocalObjectReference {
	var secret string
	var nodeSecret string

	if tserver.Spec.Release != nil {
		if tserver.Spec.Release.Secret != "" {
			secret = tserver.Spec.Release.Secret
		}

		if tserver.Spec.Tars != nil && tserver.Spec.Release.TServerReleaseNode != nil {
			nodeSecret = tserver.Spec.Release.TServerReleaseNode.Secret
		}
	}

	var secrets []k8sCoreV1.LocalObjectReference
	if secret != "" {
		secrets = append(secrets, k8sCoreV1.LocalObjectReference{
			Name: secret,
		})
	}

	if nodeSecret != "" && nodeSecret != secret {
		secrets = append(secrets, k8sCoreV1.LocalObjectReference{
			Name: nodeSecret,
		})
	}
	return secrets
}

func buildDaemonsetUpdateStrategy(tserver *crdV1beta2.TServer) k8sAppsV1.DaemonSetUpdateStrategy {
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

func buildStatefulsetUpdateStrategy(tserver *crdV1beta2.TServer) k8sAppsV1.StatefulSetUpdateStrategy {
	return tserver.Spec.K8S.UpdateStrategy
}

func buildContainerPorts(tserver *crdV1beta2.TServer) []k8sCoreV1.ContainerPort {

	var containerPorts []k8sCoreV1.ContainerPort

	var tserverPorts = map[string]*crdV1beta2.TServerPort{}
	var tserverServants = map[string]*crdV1beta2.TServerServant{}
	var tk8sHostPorts = map[string]*crdV1beta2.TK8SHostPort{}

	if tserver.Spec.Tars != nil {
		for _, servant := range tserver.Spec.Tars.Servants {
			tserverServants[servant.Name] = servant
		}
		for _, port := range tserver.Spec.Tars.Ports {
			tserverPorts[port.Name] = port
		}
	} else if tserver.Spec.Normal != nil {
		for _, port := range tserver.Spec.Normal.Ports {
			tserverPorts[port.Name] = port
		}
	}

	if !tserver.Spec.K8S.HostNetwork {
		for _, hostPort := range tserver.Spec.K8S.HostPorts {
			tk8sHostPorts[hostPort.NameRef] = hostPort
		}
	}

	getProtocol := func(isTcp bool) k8sCoreV1.Protocol {
		if isTcp {
			return k8sCoreV1.ProtocolTCP
		}
		return k8sCoreV1.ProtocolUDP
	}

	for k, v := range tserverPorts {
		if hostPort, ok := tk8sHostPorts[k]; ok {
			containerPorts = append(containerPorts, k8sCoreV1.ContainerPort{
				Name:          v.Name,
				ContainerPort: v.Port,
				Protocol:      getProtocol(v.IsTcp),
				HostPort:      hostPort.Port,
			})
		} else {
			containerPorts = append(containerPorts, k8sCoreV1.ContainerPort{
				Name:          v.Name,
				ContainerPort: v.Port,
				Protocol:      getProtocol(v.IsTcp),
			})
		}
	}

	for k, v := range tserverServants {
		if hostPort, ok := tk8sHostPorts[k]; ok {
			containerPorts = append(containerPorts, k8sCoreV1.ContainerPort{
				Name:          fmt.Sprintf("p%d-%d", hostPort.Port, v.Port),
				ContainerPort: v.Port,
				HostPort:      hostPort.Port,
				Protocol:      getProtocol(v.IsTcp),
			})
		}
	}
	return containerPorts
}

func buildPodReadinessGates(tserver *crdV1beta2.TServer) []k8sCoreV1.PodReadinessGate {
	if tserver.Spec.K8S.ReadinessGate == nil || *tserver.Spec.K8S.ReadinessGate == "" {
		return nil
	}
	return []k8sCoreV1.PodReadinessGate{
		{
			ConditionType: crdMeta.TPodReadinessGate,
		},
	}
}

func buildPodAffinity(tserver *crdV1beta2.TServer) *k8sCoreV1.Affinity {
	var nodeSelectorTerm []k8sCoreV1.NodeSelectorRequirement
	for _, selector := range tserver.Spec.K8S.NodeSelector {
		nodeSelectorTerm = append(nodeSelectorTerm, selector)
	}

	nodeSelectorTerm = append(nodeSelectorTerm,
		k8sCoreV1.NodeSelectorRequirement{
			Key:      fmt.Sprintf("%s.%s", crdMeta.TarsNodeLabel, tserver.Namespace),
			Operator: k8sCoreV1.NodeSelectorOpExists,
		},
	)

	var podAntiAffinity *k8sCoreV1.PodAntiAffinity
	var preferredSchedulingTerms []k8sCoreV1.PreferredSchedulingTerm

	if !tserver.Spec.K8S.DaemonSet {
		switch tserver.Spec.K8S.AbilityAffinity {
		case crdV1beta2.AppRequired:
			nodeSelectorTerm = append(nodeSelectorTerm,
				k8sCoreV1.NodeSelectorRequirement{
					Key:      fmt.Sprintf("%s.%s.%s", crdMeta.TarsAbilityLabelPrefix, tserver.Namespace, tserver.Spec.App),
					Operator: k8sCoreV1.NodeSelectorOpExists,
				},
			)
		case crdV1beta2.ServerRequired:
			nodeSelectorTerm = append(nodeSelectorTerm,
				k8sCoreV1.NodeSelectorRequirement{
					Key:      fmt.Sprintf("%s.%s.%s-%s", crdMeta.TarsAbilityLabelPrefix, tserver.Namespace, tserver.Spec.App, tserver.Spec.Server),
					Operator: k8sCoreV1.NodeSelectorOpExists,
				},
			)
		case crdV1beta2.AppOrServerPreferred:
			preferredSchedulingTerms = []k8sCoreV1.PreferredSchedulingTerm{
				{
					Weight: 60,
					Preference: k8sCoreV1.NodeSelectorTerm{
						MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
							{
								Key:      fmt.Sprintf("%s.%s.%s-%s", crdMeta.TarsAbilityLabelPrefix, tserver.Namespace, tserver.Spec.App, tserver.Spec.Server),
								Operator: k8sCoreV1.NodeSelectorOpExists,
							},
						},
					},
				},
				{
					Weight: 30,
					Preference: k8sCoreV1.NodeSelectorTerm{
						MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
							{
								Key:      fmt.Sprintf("%s.%s.%s", crdMeta.TarsAbilityLabelPrefix, tserver.Namespace, tserver.Spec.App),
								Operator: k8sCoreV1.NodeSelectorOpExists,
							},
						},
					},
				},
			}
		case crdV1beta2.None:
		}
		if tserver.Spec.K8S.NotStacked {
			podAntiAffinity = &k8sCoreV1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PodAffinityTerm{
					{
						LabelSelector: &k8sMetaV1.LabelSelector{
							MatchLabels: map[string]string{
								crdMeta.TServerAppLabel:  tserver.Spec.App,
								crdMeta.TServerNameLabel: tserver.Spec.Server,
							},
						},
						Namespaces:  []string{tserver.Namespace},
						TopologyKey: crdMeta.TServerNameLabel,
					},
				},
			}
		}
	}

	affinity := &k8sCoreV1.Affinity{
		NodeAffinity: &k8sCoreV1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
				{
					MatchExpressions: nodeSelectorTerm,
				},
			}},
			PreferredDuringSchedulingIgnoredDuringExecution: preferredSchedulingTerms,
		},
		PodAntiAffinity: podAntiAffinity,
	}
	return affinity
}

func buildPodTemplate(tserver *crdV1beta2.TServer) k8sCoreV1.PodTemplateSpec {
	var enableServiceLinks = false
	var FixedDNSConfigNDOTS = "2"

	var dnsPolicy = k8sCoreV1.DNSClusterFirst
	if tserver.Spec.K8S.HostNetwork {
		dnsPolicy = k8sCoreV1.DNSClusterFirstWithHostNet
	}

	serverImage := crdMeta.ServiceImagePlaceholder

	if tserver.Spec.Release != nil {
		serverImage = tserver.Spec.Release.Image
	}

	spec := k8sCoreV1.PodTemplateSpec{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: tserver.Name,
			Labels: map[string]string{
				crdMeta.TServerAppLabel:  tserver.Spec.App,
				crdMeta.TServerNameLabel: tserver.Spec.Server,
			},
		},
		Spec: k8sCoreV1.PodSpec{
			Volumes:        buildPodVolumes(tserver),
			InitContainers: buildPodInitContainers(tserver),
			Containers: []k8sCoreV1.Container{
				{
					Name:            tserver.Name,
					Image:           serverImage,
					Ports:           buildContainerPorts(tserver),
					EnvFrom:         tserver.Spec.K8S.EnvFrom,
					Env:             tserver.Spec.K8S.Env,
					Resources:       tserver.Spec.K8S.Resources,
					VolumeMounts:    buildContainerVolumeMounts(tserver),
					ImagePullPolicy: tserver.Spec.K8S.ImagePullPolicy,
				},
			},
			RestartPolicy:      k8sCoreV1.RestartPolicyAlways,
			DNSPolicy:          dnsPolicy,
			ServiceAccountName: tserver.Spec.K8S.ServiceAccount,
			HostNetwork:        tserver.Spec.K8S.HostNetwork,
			HostPID:            false,
			HostIPC:            tserver.Spec.K8S.HostIPC,
			ImagePullSecrets:   buildPodImagePullSecrets(tserver),
			Affinity:           buildPodAffinity(tserver),
			DNSConfig: &k8sCoreV1.PodDNSConfig{
				Options: []k8sCoreV1.PodDNSConfigOption{
					{
						Name:  "ndots",
						Value: &FixedDNSConfigNDOTS,
					},
				},
			},
			ReadinessGates:     buildPodReadinessGates(tserver),
			EnableServiceLinks: &enableServiceLinks,
		},
	}

	if tserver.Spec.Release != nil {
		spec.Labels[crdMeta.TServerIdLabel] = tserver.Spec.Release.ID
	}

	return spec
}

func buildTVolumeClaimTemplates(tserver *crdV1beta2.TServer, name string) *k8sCoreV1.PersistentVolumeClaim {
	storageClassName := crdMeta.TStorageClassName
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
				crdMeta.TServerAppLabel:   tserver.Spec.App,
				crdMeta.TServerNameLabel:  tserver.Spec.Server,
				crdMeta.TLocalVolumeLabel: name,
			},
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
			},
		},
		Spec: k8sCoreV1.PersistentVolumeClaimSpec{
			AccessModes: []k8sCoreV1.PersistentVolumeAccessMode{k8sCoreV1.ReadWriteOnce},
			Selector: &k8sMetaV1.LabelSelector{
				MatchLabels: map[string]string{
					crdMeta.TServerAppLabel:   tserver.Spec.App,
					crdMeta.TServerNameLabel:  tserver.Spec.Server,
					crdMeta.TLocalVolumeLabel: name,
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

func buildStatefulsetVolumeClaimTemplates(tserver *crdV1beta2.TServer) []k8sCoreV1.PersistentVolumeClaim {
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
		volumeClaimTemplates = append(volumeClaimTemplates, *buildTVolumeClaimTemplates(tserver, crdMeta.THostBindPlaceholder))
	}

	return volumeClaimTemplates
}

func buildStatefulset(tserver *crdV1beta2.TServer) *k8sAppsV1.StatefulSet {
	var statefulSet = &k8sAppsV1.StatefulSet{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserver.Name,
			Namespace: tserver.Namespace,
			Labels: map[string]string{
				crdMeta.TServerAppLabel:  tserver.Spec.App,
				crdMeta.TServerNameLabel: tserver.Spec.Server,
			},
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
			},
		},
		Spec: k8sAppsV1.StatefulSetSpec{
			Replicas: &tserver.Spec.K8S.Replicas,
			Selector: &k8sMetaV1.LabelSelector{
				MatchLabels: map[string]string{
					crdMeta.TServerAppLabel:  tserver.Spec.App,
					crdMeta.TServerNameLabel: tserver.Spec.Server,
				},
			},
			Template:             buildPodTemplate(tserver),
			VolumeClaimTemplates: buildStatefulsetVolumeClaimTemplates(tserver),
			ServiceName:          tserver.Name,
			PodManagementPolicy:  tserver.Spec.K8S.PodManagementPolicy,
			UpdateStrategy:       buildStatefulsetUpdateStrategy(tserver),
		},
	}
	return statefulSet
}

func buildDaemonset(tserver *crdV1beta2.TServer) *k8sAppsV1.DaemonSet {
	daemonSet := &k8sAppsV1.DaemonSet{
		TypeMeta: k8sMetaV1.TypeMeta{},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserver.Name,
			Namespace: tserver.Namespace,
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
			},
			Labels: map[string]string{
				crdMeta.TServerAppLabel:  tserver.Spec.App,
				crdMeta.TServerNameLabel: tserver.Spec.Server,
			},
		},
		Spec: k8sAppsV1.DaemonSetSpec{
			Selector: &k8sMetaV1.LabelSelector{
				MatchLabels: map[string]string{
					crdMeta.TServerAppLabel:  tserver.Spec.App,
					crdMeta.TServerNameLabel: tserver.Spec.Server,
				},
			},
			Template:       buildPodTemplate(tserver),
			UpdateStrategy: buildDaemonsetUpdateStrategy(tserver),
		},
	}
	return daemonSet
}

func buildServicePorts(tserver *crdV1beta2.TServer) []k8sCoreV1.ServicePort {
	var ports []k8sCoreV1.ServicePort

	getProtocol := func(isTcp bool) k8sCoreV1.Protocol {
		if isTcp {
			return k8sCoreV1.ProtocolTCP
		}
		return k8sCoreV1.ProtocolUDP
	}

	if tserver.Spec.Tars != nil {
		for _, v := range tserver.Spec.Tars.Servants {
			ports = append(ports, k8sCoreV1.ServicePort{
				Name:       strings.ToLower(v.Name),
				Protocol:   getProtocol(v.IsTcp),
				Port:       v.Port,
				TargetPort: intstr.FromInt(int(v.Port)),
			})
		}
		for _, v := range tserver.Spec.Tars.Ports {
			ports = append(ports, k8sCoreV1.ServicePort{
				Name:       strings.ToLower(v.Name),
				Protocol:   getProtocol(v.IsTcp),
				Port:       v.Port,
				TargetPort: intstr.FromInt(int(v.Port)),
			})
		}
	} else if tserver.Spec.Normal != nil {
		for _, v := range tserver.Spec.Normal.Ports {
			ports = append(ports, k8sCoreV1.ServicePort{
				Name:       strings.ToLower(v.Name),
				Protocol:   getProtocol(v.IsTcp),
				Port:       v.Port,
				TargetPort: intstr.FromInt(int(v.Port)),
			})
		}
	}
	return ports
}

func buildService(tserver *crdV1beta2.TServer) *k8sCoreV1.Service {
	service := &k8sCoreV1.Service{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserver.Name,
			Namespace: tserver.Namespace,
			Labels: map[string]string{
				crdMeta.TServerAppLabel:  tserver.Spec.App,
				crdMeta.TServerNameLabel: tserver.Spec.Server,
			},
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
			},
		},
		Spec: k8sCoreV1.ServiceSpec{
			Selector: map[string]string{
				crdMeta.TServerAppLabel:  tserver.Spec.App,
				crdMeta.TServerNameLabel: tserver.Spec.Server,
			},
			ClusterIP: k8sCoreV1.ClusterIPNone,
			Type:      k8sCoreV1.ServiceTypeClusterIP,
			Ports:     buildServicePorts(tserver),
		},
	}
	return service
}

func buildTEndpoint(tserver *crdV1beta2.TServer) *crdV1beta2.TEndpoint {
	endpoint := &crdV1beta2.TEndpoint{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserver.Name,
			Namespace: tserver.Namespace,
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
			},
			Labels: map[string]string{
				crdMeta.TServerAppLabel:  tserver.Spec.App,
				crdMeta.TServerNameLabel: tserver.Spec.Server,
			},
		},
		Spec: crdV1beta2.TEndpointSpec{
			App:       tserver.Spec.App,
			Server:    tserver.Spec.Server,
			SubType:   tserver.Spec.SubType,
			Important: tserver.Spec.Important,
			Tars:       tserver.Spec.Tars,
			Normal:    tserver.Spec.Normal,
			HostPorts: tserver.Spec.K8S.HostPorts,
			Release:   tserver.Spec.Release,
		},
	}
	return endpoint
}

func buildTExitedRecord(tserver *crdV1beta2.TServer) *crdV1beta2.TExitedRecord {
	tExitedRecord := &crdV1beta2.TExitedRecord{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserver.Name,
			Namespace: tserver.Namespace,
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
			},
			Labels: map[string]string{
				crdMeta.TServerAppLabel:  tserver.Spec.App,
				crdMeta.TServerNameLabel: tserver.Spec.Server,
			},
		},
		App:    tserver.Spec.App,
		Server: tserver.Spec.Server,
		Pods:   []crdV1beta2.TExitedPod{},
	}
	return tExitedRecord
}

func syncTEndpoint(tserver *crdV1beta2.TServer, endpoint *crdV1beta2.TEndpoint) {
	endpoint.Labels = tserver.Labels
	endpoint.OwnerReferences = []k8sMetaV1.OwnerReference{
		*k8sMetaV1.NewControllerRef(tserver, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
	}
	endpoint.Spec.App = tserver.Spec.App
	endpoint.Spec.Server = tserver.Spec.Server
	endpoint.Spec.SubType = tserver.Spec.SubType
	endpoint.Spec.Important = tserver.Spec.Important
	endpoint.Spec.Tars = tserver.Spec.Tars
	endpoint.Spec.Normal = tserver.Spec.Normal
	endpoint.Spec.HostPorts = tserver.Spec.K8S.HostPorts
	endpoint.Spec.Release = tserver.Spec.Release
}

func syncService(tserver *crdV1beta2.TServer, service *k8sCoreV1.Service) {
	service.Spec.Ports = buildServicePorts(tserver)
}

func syncStatefulSet(tserver *crdV1beta2.TServer, statefulSet *k8sAppsV1.StatefulSet) {

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

func syncDaemonSet(tserver *crdV1beta2.TServer, daemonSet *k8sAppsV1.DaemonSet) {
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
