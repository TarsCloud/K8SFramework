package v1beta3

import (
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
)

func buildTEndpoint(tserver *tarsV1beta3.TServer) *tarsV1beta3.TEndpoint {
	tendpoint := &tarsV1beta3.TEndpoint{
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
		Spec: tarsV1beta3.TEndpointSpec{
			App:       tserver.Spec.App,
			Server:    tserver.Spec.Server,
			SubType:   tserver.Spec.SubType,
			Important: tserver.Spec.Important,
			Tars:      tserver.Spec.Tars,
			Normal:    tserver.Spec.Normal,
			HostPorts: tserver.Spec.K8S.HostPorts,
			Release:   tserver.Spec.Release,
		},
	}
	return tendpoint
}

func syncTEndpoint(tserver *tarsV1beta3.TServer, tendpoint *tarsV1beta3.TEndpoint) {
	tendpoint.Labels = tserver.Labels
	tendpoint.OwnerReferences = []k8sMetaV1.OwnerReference{
		*k8sMetaV1.NewControllerRef(tserver, tarsV1beta3.SchemeGroupVersion.WithKind(tarsMeta.TServerKind)),
	}
	tendpoint.Spec.App = tserver.Spec.App
	tendpoint.Spec.Server = tserver.Spec.Server
	tendpoint.Spec.SubType = tserver.Spec.SubType
	tendpoint.Spec.Important = tserver.Spec.Important
	tendpoint.Spec.Tars = tserver.Spec.Tars
	tendpoint.Spec.Normal = tserver.Spec.Normal
	tendpoint.Spec.HostPorts = tserver.Spec.K8S.HostPorts
	tendpoint.Spec.Release = tserver.Spec.Release
}
