package v1beta3

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	"k8s.tars.io/translator"
)

type Translator struct {
}

var runtimeConfig translator.RunTimeConfig

func NewTranslator(rc translator.RunTimeConfig) *Translator {
	runtimeConfig = rc
	return &Translator{}
}

func (*Translator) BuildService(tserver *tarsV1beta3.TServer) *k8sCoreV1.Service {
	return buildService(tserver)
}

func (*Translator) BuildStatefulset(tserver *tarsV1beta3.TServer) *k8sAppsV1.StatefulSet {
	return buildStatefulset(tserver)
}

func (*Translator) BuildStatefulsetVolumeClaimTemplates(tserver *tarsV1beta3.TServer) []k8sCoreV1.PersistentVolumeClaim {
	return buildStatefulsetVolumeClaimTemplates(tserver)
}

func (*Translator) BuildDaemonset(tserver *tarsV1beta3.TServer) *k8sAppsV1.DaemonSet {
	return buildDaemonset(tserver)
}

func (*Translator) BuildTEndpoint(tserver *tarsV1beta3.TServer) *tarsV1beta3.TEndpoint {
	return buildTEndpoint(tserver)
}

func (*Translator) BuildTExitedRecord(tserver *tarsV1beta3.TServer) *tarsV1beta3.TExitedRecord {
	return buildTExitedRecord(tserver)
}

func (*Translator) DryRunSyncService(tserver *tarsV1beta3.TServer, service *k8sCoreV1.Service) (bool, *k8sCoreV1.Service) {
	if !equalTServerAndService(tserver, service) {
		cp := service.DeepCopy()
		syncService(tserver, cp)
		return true, cp
	}
	return false, nil
}

func (*Translator) DryRunSyncStatefulset(tserver *tarsV1beta3.TServer, statefulset *k8sAppsV1.StatefulSet) (bool, *k8sAppsV1.StatefulSet) {
	if !equalTServerAndStatefulset(tserver, statefulset) {
		cp := statefulset.DeepCopy()
		syncStatefulSet(tserver, cp)
		return true, cp
	}

	return false, nil
}

func (*Translator) DryRunSyncDaemonset(tserver *tarsV1beta3.TServer, daemonset *k8sAppsV1.DaemonSet) (bool, *k8sAppsV1.DaemonSet) {
	if !equalTServerAndDaemonSet(tserver, daemonset) {
		cp := daemonset.DeepCopy()
		syncDaemonSet(tserver, cp)
		return true, cp
	}
	return false, nil
}

func (*Translator) DryRunSyncTEndpoint(tserver *tarsV1beta3.TServer, tendpoint *tarsV1beta3.TEndpoint) (bool, *tarsV1beta3.TEndpoint) {
	if !equalTServerAndTEndpoint(tserver, tendpoint) {
		cp := tendpoint.DeepCopy()
		syncTEndpoint(tserver, cp)
		return true, cp
	}
	return false, nil
}
