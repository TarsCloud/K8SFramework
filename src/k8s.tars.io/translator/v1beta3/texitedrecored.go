package v1beta3

import (
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
)

func buildTExitedRecord(tserver *tarsV1beta3.TServer) *tarsV1beta3.TExitedRecord {
	tExitedRecord := &tarsV1beta3.TExitedRecord{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserver.Name,
			Namespace: tserver.Namespace,
			OwnerReferences: []k8sMetaV1.OwnerReference{
				*k8sMetaV1.NewControllerRef(tserver, tarsV1beta3.SchemeGroupVersion.WithKind(tarsMeta.TServerKind)),
			},
			Labels: map[string]string{
				tarsMeta.TServerAppLabel:  tserver.Spec.App,
				tarsMeta.TServerNameLabel: tserver.Spec.Server,
			},
		},
		App:    tserver.Spec.App,
		Server: tserver.Spec.Server,
		Pods:   []tarsV1beta3.TExitedPod{},
	}
	return tExitedRecord
}
