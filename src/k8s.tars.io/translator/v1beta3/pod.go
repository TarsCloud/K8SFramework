package v1beta3

import (
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
)

func buildContainerPorts(tserver *tarsV1beta3.TServer) []k8sCoreV1.ContainerPort {

	var containerPorts []k8sCoreV1.ContainerPort

	var tserverPorts = map[string]*tarsV1beta3.TServerPort{}
	var tserverServants = map[string]*tarsV1beta3.TServerServant{}
	var tk8sHostPorts = map[string]*tarsV1beta3.TK8SHostPort{}

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

func buildContainerVolumeMounts(tserver *tarsV1beta3.TServer) []k8sCoreV1.VolumeMount {
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

	if tserver.Spec.SubType == tarsV1beta3.TARS {
		volumeMounts = append(volumeMounts, k8sCoreV1.VolumeMount{
			Name:      "tarsnode-work-dir",
			MountPath: "/usr/local/app/tars/tarsnode",
		})
	}
	return volumeMounts
}

func buildPodReadinessGates(tserver *tarsV1beta3.TServer) []k8sCoreV1.PodReadinessGate {
	var gates []k8sCoreV1.PodReadinessGate
	for _, v := range tserver.Spec.K8S.ReadinessGates {
		gates = append(gates, k8sCoreV1.PodReadinessGate{
			ConditionType: k8sCoreV1.PodConditionType(v),
		})
	}
	return gates
}

func buildPodAffinity(tserver *tarsV1beta3.TServer) *k8sCoreV1.Affinity {
	var nodeSelectorTerm []k8sCoreV1.NodeSelectorRequirement
	for _, selector := range tserver.Spec.K8S.NodeSelector {
		nodeSelectorTerm = append(nodeSelectorTerm, selector)
	}

	nodeSelectorTerm = append(nodeSelectorTerm,
		k8sCoreV1.NodeSelectorRequirement{
			Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, tserver.Namespace),
			Operator: k8sCoreV1.NodeSelectorOpExists,
		},
	)

	var podAntiAffinity *k8sCoreV1.PodAntiAffinity
	var preferredSchedulingTerms []k8sCoreV1.PreferredSchedulingTerm

	if !tserver.Spec.K8S.DaemonSet {
		switch tserver.Spec.K8S.AbilityAffinity {
		case tarsV1beta3.AppRequired:
			nodeSelectorTerm = append(nodeSelectorTerm,
				k8sCoreV1.NodeSelectorRequirement{
					Key:      fmt.Sprintf("%s.%s.%s", tarsMeta.TarsAbilityLabelPrefix, tserver.Namespace, tserver.Spec.App),
					Operator: k8sCoreV1.NodeSelectorOpExists,
				},
			)
		case tarsV1beta3.ServerRequired:
			nodeSelectorTerm = append(nodeSelectorTerm,
				k8sCoreV1.NodeSelectorRequirement{
					Key:      fmt.Sprintf("%s.%s.%s-%s", tarsMeta.TarsAbilityLabelPrefix, tserver.Namespace, tserver.Spec.App, tserver.Spec.Server),
					Operator: k8sCoreV1.NodeSelectorOpExists,
				},
			)
		case tarsV1beta3.AppOrServerPreferred:
			preferredSchedulingTerms = []k8sCoreV1.PreferredSchedulingTerm{
				{
					Weight: 60,
					Preference: k8sCoreV1.NodeSelectorTerm{
						MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
							{
								Key:      fmt.Sprintf("%s.%s.%s-%s", tarsMeta.TarsAbilityLabelPrefix, tserver.Namespace, tserver.Spec.App, tserver.Spec.Server),
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
								Key:      fmt.Sprintf("%s.%s.%s", tarsMeta.TarsAbilityLabelPrefix, tserver.Namespace, tserver.Spec.App),
								Operator: k8sCoreV1.NodeSelectorOpExists,
							},
						},
					},
				},
			}
		case tarsV1beta3.None:
		}
		if tserver.Spec.K8S.NotStacked {
			podAntiAffinity = &k8sCoreV1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PodAffinityTerm{
					{
						LabelSelector: &k8sMetaV1.LabelSelector{
							MatchLabels: map[string]string{
								tarsMeta.TServerAppLabel:  tserver.Spec.App,
								tarsMeta.TServerNameLabel: tserver.Spec.Server,
							},
						},
						Namespaces:  []string{tserver.Namespace},
						TopologyKey: tarsMeta.K8SHostNameLabel,
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

func buildPodTemplate(tserver *tarsV1beta3.TServer) k8sCoreV1.PodTemplateSpec {
	var enableServiceLinks = false
	var fixedDNSConfigNDOTS = "2"

	var dnsPolicy = k8sCoreV1.DNSClusterFirst
	if tserver.Spec.K8S.HostNetwork {
		dnsPolicy = k8sCoreV1.DNSClusterFirstWithHostNet
	}

	serverImage := tarsMeta.ServiceImagePlaceholder

	if tserver.Spec.Release != nil {
		serverImage = tserver.Spec.Release.Image
	}

	spec := k8sCoreV1.PodTemplateSpec{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: tserver.Name,
			Labels: map[string]string{
				tarsMeta.TServerAppLabel:  tserver.Spec.App,
				tarsMeta.TServerNameLabel: tserver.Spec.Server,
			},
		},
		Spec: k8sCoreV1.PodSpec{
			Volumes:        buildPodVolumes(tserver),
			InitContainers: buildPodInitContainers(tserver),
			Containers: []k8sCoreV1.Container{
				{
					Name:            tserver.Name,
					Image:           serverImage,
					Command:         tserver.Spec.K8S.Command,
					Args:            tserver.Spec.K8S.Args,
					Ports:           buildContainerPorts(tserver),
					EnvFrom:         tserver.Spec.K8S.EnvFrom,
					Env:             tserver.Spec.K8S.Env,
					Resources:       tserver.Spec.K8S.Resources,
					VolumeMounts:    buildContainerVolumeMounts(tserver),
					ImagePullPolicy: tserver.Spec.K8S.ImagePullPolicy,
				},
			},
			EphemeralContainers: nil,
			RestartPolicy:       k8sCoreV1.RestartPolicyAlways,
			DNSPolicy:           dnsPolicy,
			ServiceAccountName:  tserver.Spec.K8S.ServiceAccount,
			HostNetwork:         tserver.Spec.K8S.HostNetwork,
			HostIPC:             tserver.Spec.K8S.HostIPC,
			ImagePullSecrets:    buildPodImagePullSecrets(tserver),
			Affinity:            buildPodAffinity(tserver),
			DNSConfig: &k8sCoreV1.PodDNSConfig{
				Options: []k8sCoreV1.PodDNSConfigOption{
					{
						Name:  "ndots",
						Value: &fixedDNSConfigNDOTS,
					},
				},
			},
			ReadinessGates:     buildPodReadinessGates(tserver),
			EnableServiceLinks: &enableServiceLinks,
		},
	}

	if tserver.Spec.Release != nil {
		spec.Labels[tarsMeta.TServerIdLabel] = tserver.Spec.Release.ID
	}

	return spec
}

func buildPodVolumes(tserver *tarsV1beta3.TServer) []k8sCoreV1.Volume {
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

	if tserver.Spec.SubType == tarsV1beta3.TARS {
		volumes = append(volumes, k8sCoreV1.Volume{
			Name: "tarsnode-work-dir",
			VolumeSource: k8sCoreV1.VolumeSource{
				EmptyDir: &k8sCoreV1.EmptyDirVolumeSource{},
			}})
	}
	return volumes
}

func buildPodInitContainers(tserver *tarsV1beta3.TServer) []k8sCoreV1.Container {
	if tserver.Spec.SubType != tarsV1beta3.TARS {
		return nil
	}

	var image string
	if tserver.Spec.Release != nil && tserver.Spec.Release.TServerReleaseNode != nil {
		image = tserver.Spec.Release.TServerReleaseNode.Image
	}

	if image == "" || image == tarsMeta.ServiceImagePlaceholder {
		image, _ = runtimeConfig.GetDefaultNodeImage(tserver.Namespace)
	}

	if image == tarsMeta.ServiceImagePlaceholder {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ShouldNotHappenError, "no node image set"))
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

	if tserver.Spec.K8S.LauncherType != tarsMeta.Background {
		containers[0].Env = append(containers[0].Env,
			k8sCoreV1.EnvVar{
				Name:  "LauncherType",
				Value: string(tserver.Spec.K8S.LauncherType),
			})
	}

	return containers
}

func buildPodImagePullSecrets(tserver *tarsV1beta3.TServer) []k8sCoreV1.LocalObjectReference {
	var secret string
	var nodeSecret string

	if tserver.Spec.Release != nil {
		if tserver.Spec.Release.Secret != "" {
			secret = tserver.Spec.Release.Secret
		}

		if tserver.Spec.Tars != nil && tserver.Spec.Release.TServerReleaseNode != nil {
			nodeSecret = tserver.Spec.Release.TServerReleaseNode.Secret
			if nodeSecret == "" {
				_, nodeSecret = runtimeConfig.GetDefaultNodeImage(tserver.Namespace)
			}
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
