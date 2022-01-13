package v1beta2

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
)

func equalServicePort(l, r []k8sCoreV1.ServicePort) bool {

	if len(l) != len(r) {
		return false
	}

	for i := range l {
		if l[i].Name != r[i].Name {
			return false
		}
		if l[i].Port != r[i].Port {
			return false
		}
		if l[i].Protocol != r[i].Protocol {
			return false
		}
		if l[i].NodePort != r[i].NodePort {
			return false
		}
		if l[i].TargetPort.Type != r[i].TargetPort.Type {
			return false
		}
		if l[i].TargetPort.IntVal != r[i].TargetPort.IntVal {
			return false
		}
		if l[i].TargetPort.StrVal != r[i].TargetPort.StrVal {
			return false
		}
	}
	return true
}

func ContainLabel(l, r map[string]string) bool {
	if len(l) > len(r) {
		return false
	}

	for lk, lv := range l {
		if rv, ok := r[lk]; !ok || rv != lv {
			return false
		}
	}

	return true
}

func equalLabel(l, r map[string]string) bool {
	if len(l) != len(r) {
		return false
	}
	for lk, lv := range l {
		if rv, ok := r[lk]; !ok || rv != lv {
			return false
		}
	}
	return true
}

func equalEnvFieldRef(l, r *k8sCoreV1.ObjectFieldSelector) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}

	if l.FieldPath != r.FieldPath {
		return false
	}
	return true
}

func equalEnvConfigMap(l, r *k8sCoreV1.ConfigMapKeySelector) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}
	if l.Key != r.Key {
		return false
	}
	if l.Name != r.Name {
		return false
	}
	return true
}

func equalEnvResourceFieldRef(l, r *k8sCoreV1.ResourceFieldSelector) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}
	if l.ContainerName != r.ContainerName {
		return false
	}
	if l.Resource != r.Resource {
		return false
	}
	if l.ContainerName != r.ContainerName {
		return false
	}
	return true
}

func equalEnvSecretKeyRef(l, r *k8sCoreV1.SecretKeySelector) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}
	if l.Key != r.Key {
		return false
	}
	if l.Name != r.Name {
		return false
	}
	return true
}

func equalEnv(l, r []k8sCoreV1.EnvVar) bool {
	if len(l) != len(r) {
		return false
	}
	for i := range l {
		if l[i].Name != r[i].Name {
			return false
		}
		if l[i].Value != r[i].Value {
			return false
		}

		if l[i].ValueFrom == nil {
			if r[i].ValueFrom == nil {
				continue
			}
			return false
		}

		if r[i].ValueFrom == nil {
			return false
		}

		if !equalEnvConfigMap(l[i].ValueFrom.ConfigMapKeyRef, r[i].ValueFrom.ConfigMapKeyRef) {
			return false
		}
		if !equalEnvFieldRef(l[i].ValueFrom.FieldRef, r[i].ValueFrom.FieldRef) {
			return false
		}
		if !equalEnvResourceFieldRef(l[i].ValueFrom.ResourceFieldRef, r[i].ValueFrom.ResourceFieldRef) {
			return false
		}
		if !equalEnvSecretKeyRef(l[i].ValueFrom.SecretKeyRef, r[i].ValueFrom.SecretKeyRef) {
			return false
		}
	}
	return true
}

func equalEnvFrom(l, r []k8sCoreV1.EnvFromSource) bool {
	if len(l) != len(r) {
		return false
	}
	for i := range l {
		if l[i].Prefix != r[i].Prefix {
			return false
		}
		if !equality.Semantic.DeepEqual(l[i].ConfigMapRef, r[i].ConfigMapRef) {
			return false
		}
		if !equality.Semantic.DeepEqual(l[i].SecretRef, r[i].SecretRef) {
			return false
		}
	}
	return true
}

func equalVolumesDownwardAPI(l, r *k8sCoreV1.DownwardAPIVolumeSource) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}

	if r == nil {
		return false
	}

	if len(l.Items) != len(r.Items) {
		return false
	}
	for i := range l.Items {
		if l.Items[i].Path != r.Items[i].Path {
			return false
		}
		if !equality.Semantic.DeepEqual(l.Items[i].Mode, r.Items[i].Mode) {
			return false
		}
	}
	return true
}

func equalVolumesPVC(l, r *k8sCoreV1.PersistentVolumeClaimVolumeSource) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}

	if l.ClaimName != r.ClaimName {
		return false
	}

	if l.ReadOnly == r.ReadOnly {
		return false
	}

	return true
}

func equalVolumesHostPath(l, r *k8sCoreV1.HostPathVolumeSource) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}
	if l.Path != r.Path {
		return false
	}

	if l.Type == nil || *l.Type == "" {
		if r.Type != nil && *r.Type != "" {
			return false
		}
	} else {
		if *l.Type != *r.Type {
			return false
		}
	}
	return true
}

func equalVolumesEmptyDir(l, r *k8sCoreV1.EmptyDirVolumeSource) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}

	if l.Medium != r.Medium {
		return false
	}

	if !equality.Semantic.DeepEqual(l.SizeLimit, r.SizeLimit) {
		return false
	}

	return true
}

func equalVolumesSecret(l, r *k8sCoreV1.SecretVolumeSource) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}
	if len(l.Items) != len(r.Items) {
		return false
	}
	for i := range l.Items {
		if l.Items[i].Key != r.Items[i].Key {
			return false
		}
		if l.Items[i].Path != r.Items[i].Path {
			return false
		}
		if equality.Semantic.DeepEqual(l.Items[i].Mode, r.Items[i].Mode) {
			return false
		}
	}
	return true
}

func equalVolumesConfigMap(l, r *k8sCoreV1.ConfigMapVolumeSource) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}
	if r == nil {
		return false
	}
	for i := range l.Items {
		if l.Items[i].Key != r.Items[i].Key {
			return false
		}
		if l.Items[i].Path != r.Items[i].Path {
			return false
		}
		if equality.Semantic.DeepEqual(l.Items[i].Mode, r.Items[i].Mode) {
			return false
		}
	}
	return true
}

func equalVolumes(l, r []k8sCoreV1.Volume) bool {
	if len(l) != len(r) {
		return false
	}

	for i := range l {
		if l[i].Name != r[i].Name {
			return false
		}
		if !equalVolumesConfigMap(l[i].ConfigMap, r[i].ConfigMap) {
			return false
		}
		if !equalVolumesSecret(l[i].Secret, r[i].Secret) {
			return false
		}
		if !equalVolumesEmptyDir(l[i].EmptyDir, r[i].EmptyDir) {
			return false
		}
		if !equalVolumesHostPath(l[i].HostPath, r[i].HostPath) {
			return false
		}
		if !equalVolumesPVC(l[i].PersistentVolumeClaim, r[i].PersistentVolumeClaim) {
			return false
		}
		if !equalVolumesDownwardAPI(l[i].DownwardAPI, r[i].DownwardAPI) {
			return false
		}
	}
	return true
}

func equalVolumesMounts(l, r []k8sCoreV1.VolumeMount) bool {
	if len(l) != len(r) {
		return false
	}
	for i := range l {
		if l[i].Name != r[i].Name {
			return false
		}
		if l[i].MountPath != r[i].MountPath {
			return false
		}
		if l[i].SubPath != r[i].SubPath {
			return false
		}
		if l[i].SubPathExpr != r[i].SubPathExpr {
			return false
		}
		if l[i].ReadOnly != r[i].ReadOnly {
			return false
		}
		if !equality.Semantic.DeepEqual(l[i].MountPropagation, r[i].MountPropagation) {
			return false
		}
	}
	return true
}

func equalContainerPorts(l, r []k8sCoreV1.ContainerPort) bool {
	if len(l) != len(r) {
		return false
	}
	for i := range l {
		if l[i].Name != r[i].Name {
			return false
		}
		if l[i].Protocol != r[i].Protocol {
			return false
		}
		if l[i].HostPort != r[i].HostPort {
			return false
		}
		if l[i].HostIP != r[i].HostIP {
			return false
		}
		if l[i].ContainerPort != r[i].ContainerPort {
			return false
		}
	}
	return true
}

func equalTarsServants(l, r []*crdV1beta2.TServerServant) bool {
	if len(l) != len(r) {
		return false
	}

	for i := range l {
		if l[i].Name != r[i].Name {
			return false
		}
		if l[i].Timeout != r[i].Timeout {
			return false
		}
		if l[i].Connection != r[i].Connection {
			return false
		}
		if l[i].Thread != r[i].Thread {
			return false
		}
		if l[i].Port != r[i].Port {
			return false
		}
		if l[i].IsTars != r[i].IsTars {
			return false
		}
		if l[i].IsTcp != r[i].IsTcp {
			return false
		}
	}

	return true
}

func equalTars(l, r *crdV1beta2.TServerTars) bool {

	if l == nil {
		if r == nil {
			return true
		}
		return false
	}

	if r == nil {
		return false
	}

	if l.AsyncThread != r.AsyncThread {
		return false
	}

	if l.Profile != r.Profile {
		return false
	}

	if l.Template != r.Template {
		return false
	}

	if !equalTarsServants(l.Servants, r.Servants) {
		return false
	}

	if !equalTServerPorts(l.Ports, r.Ports) {
		return false
	}

	return true
}

func equalTServerPorts(l, r []*crdV1beta2.TServerPort) bool {

	if len(l) != len(r) {
		return false
	}

	for i := range l {
		if l[i].Name != r[i].Name {
			return false
		}
		if l[i].IsTcp != r[i].IsTcp {
			return false
		}
	}
	return true
}

func equalNormal(l, r *crdV1beta2.TServerNormal) bool {
	if l == nil {
		if r == nil {
			return true
		}
		return false
	}

	if r == nil {
		return false
	}

	if !equalTServerPorts(l.Ports, r.Ports) {
		return false
	}

	return true
}

func equalK8SHostPorts(l, r []*crdV1beta2.TK8SHostPort) bool {
	if len(l) != len(r) {
		return false
	}
	for i := range l {
		if l[i].Port != r[i].Port {
			return false
		}
		if l[i].NameRef != r[i].NameRef {
			return false
		}
	}
	return true
}

func EqualTServerAndTEndpoint(tserver *crdV1beta2.TServer, endpoint *crdV1beta2.TEndpoint) bool {

	if tserver.Spec.App != endpoint.Spec.App {
		return false
	}

	if tserver.Spec.Server != endpoint.Spec.Server {
		return false
	}

	if tserver.Spec.Important != endpoint.Spec.Important {
		return false
	}

	if !equalK8SHostPorts(tserver.Spec.K8S.HostPorts, endpoint.Spec.HostPorts) {
		return false
	}

	if !equality.Semantic.DeepEqual(tserver.Spec.Release, endpoint.Spec.Release) {
		return false
	}

	switch tserver.Spec.SubType {
	case crdV1beta2.TARS:
		return equalTars(tserver.Spec.Tars, endpoint.Spec.Tars)
	case crdV1beta2.Normal:
		return equalNormal(tserver.Spec.Normal, endpoint.Spec.Normal)
	}

	//should not reach here
	return false
}

func EqualTServerAndService(tserver *crdV1beta2.TServer, service *k8sCoreV1.Service) bool {
	tserverSpec := &tserver.Spec
	serviceSpec := &service.Spec

	if serviceSpec.Type != k8sCoreV1.ServiceTypeClusterIP || serviceSpec.ClusterIP != k8sCoreV1.ClusterIPNone {
		return false
	}

	targetLabels := map[string]string{
		crdMeta.TServerAppLabel:  tserverSpec.App,
		crdMeta.TServerNameLabel: tserverSpec.Server,
	}

	if !ContainLabel(targetLabels, service.Labels) {
		return false
	}

	if !equalLabel(targetLabels, serviceSpec.Selector) {
		return false
	}

	targetPorts := buildServicePorts(tserver)

	if !equalServicePort(targetPorts, serviceSpec.Ports) {
		return false
	}
	return true
}

func EqualTServerAndDaemonSet(tserver *crdV1beta2.TServer, daemonSet *k8sAppsV1.DaemonSet) bool {

	targetLabels := map[string]string{
		crdMeta.TServerAppLabel:  tserver.Spec.App,
		crdMeta.TServerNameLabel: tserver.Spec.Server,
	}

	if !ContainLabel(targetLabels, daemonSet.Labels) {
		return false
	}

	targetMatchLabels := targetLabels
	if !equalLabel(targetMatchLabels, daemonSet.Spec.Selector.MatchLabels) {
		return false
	}

	targetTemplateLabels := targetMatchLabels

	if tserver.Spec.Release != nil {
		targetTemplateLabels[crdMeta.TServerIdLabel] = tserver.Spec.Release.ID
	}

	if !ContainLabel(targetTemplateLabels, daemonSet.Spec.Template.Labels) {
		return false
	}

	targetUpdateStrategy := buildDaemonsetUpdateStrategy(tserver)
	if !equality.Semantic.DeepEqual(targetUpdateStrategy, daemonSet.Spec.UpdateStrategy) {
		return false
	}

	daemonsetSpecTemplateSpec := &daemonSet.Spec.Template.Spec

	if tserver.Spec.K8S.HostIPC != daemonsetSpecTemplateSpec.HostIPC {
		return false
	}

	if tserver.Spec.K8S.HostNetwork != daemonsetSpecTemplateSpec.HostNetwork {
		return false
	}

	if tserver.Spec.K8S.ServiceAccount != daemonsetSpecTemplateSpec.ServiceAccountName {
		return false
	}

	targetImagePullSecrets := buildPodImagePullSecrets(tserver)
	if !equality.Semantic.DeepEqual(targetImagePullSecrets, daemonsetSpecTemplateSpec.ImagePullSecrets) {
		return false
	}

	targetAffinity := buildPodAffinity(tserver)
	if !equality.Semantic.DeepEqual(targetAffinity, daemonsetSpecTemplateSpec.Affinity) {
		return false
	}

	targetVolumes := buildPodVolumes(tserver)
	if !equalVolumes(targetVolumes, daemonsetSpecTemplateSpec.Volumes) {
		return false
	}

	var initContainer *k8sCoreV1.Container = nil
	for _, v := range daemonsetSpecTemplateSpec.InitContainers {
		if v.Name == "tarsnode" {
			initContainer = &v
		}
	}

	if tserver.Spec.SubType == crdV1beta2.TARS && initContainer == nil {
		return false
	}

	if tserver.Spec.SubType == crdV1beta2.Normal && initContainer != nil {
		return false
	}

	var container *k8sCoreV1.Container = nil

	for _, v := range daemonsetSpecTemplateSpec.Containers {
		if v.Name == tserver.Name {
			container = &v
		}
	}

	if initContainer != nil {
		launcherType := string(crdV1beta2.Background)
		for _, e := range initContainer.Env {
			if e.Name == "LauncherType" {
				launcherType = e.Value
			}
		}
		if string(tserver.Spec.K8S.LauncherType) != launcherType {
			return false
		}
	}

	if container == nil {
		return false
	}

	targetVolumeMounts := buildContainerVolumeMounts(tserver)
	if !equalVolumesMounts(targetVolumeMounts, container.VolumeMounts) {
		return false
	}

	serverImage := crdMeta.ServiceImagePlaceholder
	if tserver.Spec.Release != nil {
		serverImage = tserver.Spec.Release.Image
	}

	if serverImage != container.Image {
		return false
	}

	if tserver.Spec.K8S.ImagePullPolicy != container.ImagePullPolicy {
		return false
	}

	if !equalEnv(tserver.Spec.K8S.Env, container.Env) {
		return false
	}

	if !equalEnvFrom(tserver.Spec.K8S.EnvFrom, container.EnvFrom) {
		return false
	}

	if !equality.Semantic.DeepEqual(tserver.Spec.K8S.Resources, container.Resources) {
		return false
	}

	targetContainerPorts := buildContainerPorts(tserver)
	if !equalContainerPorts(targetContainerPorts, container.Ports) {
		return false
	}
	return true
}

func EqualTServerAndStatefulSet(tserver *crdV1beta2.TServer, statefulSet *k8sAppsV1.StatefulSet) bool {

	if tserver.Spec.K8S.Replicas != *statefulSet.Spec.Replicas {
		return false
	}

	targetLabels := map[string]string{
		crdMeta.TServerAppLabel:  tserver.Spec.App,
		crdMeta.TServerNameLabel: tserver.Spec.Server,
	}

	if !ContainLabel(targetLabels, statefulSet.Labels) {
		return false
	}

	targetMatchLabels := targetLabels
	if !equalLabel(targetMatchLabels, statefulSet.Spec.Selector.MatchLabels) {
		return false
	}

	targetTemplateLabels := targetMatchLabels

	if tserver.Spec.Release != nil {
		targetTemplateLabels[crdMeta.TServerIdLabel] = tserver.Spec.Release.ID
	}

	if !ContainLabel(targetTemplateLabels, statefulSet.Spec.Template.Labels) {
		return false
	}

	targetUpdateStrategy := buildStatefulsetUpdateStrategy(tserver)
	if !equality.Semantic.DeepEqual(targetUpdateStrategy, statefulSet.Spec.UpdateStrategy) {
		return false
	}

	statefulSetSpecTemplateSpec := &statefulSet.Spec.Template.Spec

	if tserver.Spec.K8S.HostIPC != statefulSetSpecTemplateSpec.HostIPC {
		return false
	}

	if tserver.Spec.K8S.HostNetwork != statefulSetSpecTemplateSpec.HostNetwork {
		return false
	}

	if tserver.Spec.K8S.ServiceAccount != statefulSetSpecTemplateSpec.ServiceAccountName {
		return false
	}

	targetImagePullSecrets := buildPodImagePullSecrets(tserver)
	if !equality.Semantic.DeepEqual(targetImagePullSecrets, statefulSetSpecTemplateSpec.ImagePullSecrets) {
		return false
	}

	targetAffinity := buildPodAffinity(tserver)
	if !equality.Semantic.DeepEqual(targetAffinity, statefulSetSpecTemplateSpec.Affinity) {
		return false
	}

	targetVolumes := buildPodVolumes(tserver)
	if !equalVolumes(targetVolumes, statefulSetSpecTemplateSpec.Volumes) {
		return false
	}

	var initContainer *k8sCoreV1.Container = nil
	for _, v := range statefulSetSpecTemplateSpec.InitContainers {
		if v.Name == "tarsnode" {
			initContainer = &v
		}
	}

	if tserver.Spec.SubType == crdV1beta2.TARS && initContainer == nil {
		return false
	}

	if tserver.Spec.SubType == crdV1beta2.Normal && initContainer != nil {
		return false
	}

	var container *k8sCoreV1.Container = nil

	for _, v := range statefulSetSpecTemplateSpec.Containers {
		if v.Name == tserver.Name {
			container = &v
		}
	}

	if initContainer != nil {
		launcherType := string(crdV1beta2.Background)
		for _, e := range initContainer.Env {
			if e.Name == "LauncherType" {
				launcherType = e.Value
			}
		}
		if string(tserver.Spec.K8S.LauncherType) != launcherType {
			return false
		}

		if tserver.Spec.Release != nil && tserver.Spec.Release.TServerReleaseNode != nil {
			if tserver.Spec.Release.TServerReleaseNode.Image != initContainer.Image {
				return false
			}
		}
	}

	if container == nil {
		return false
	}

	targetVolumeMounts := buildContainerVolumeMounts(tserver)
	if !equalVolumesMounts(targetVolumeMounts, container.VolumeMounts) {
		return false
	}

	serverImage := crdMeta.ServiceImagePlaceholder
	if tserver.Spec.Release != nil {
		serverImage = tserver.Spec.Release.Image
	}

	if serverImage != container.Image {
		return false
	}

	if tserver.Spec.K8S.ImagePullPolicy != container.ImagePullPolicy {
		return false
	}

	if !equalEnv(tserver.Spec.K8S.Env, container.Env) {
		return false
	}

	if !equalEnvFrom(tserver.Spec.K8S.EnvFrom, container.EnvFrom) {
		return false
	}

	if !equality.Semantic.DeepEqual(tserver.Spec.K8S.Resources, container.Resources) {
		return false
	}

	targetContainerPorts := buildContainerPorts(tserver)
	if !equalContainerPorts(targetContainerPorts, container.Ports) {
		return false
	}
	return true
}
