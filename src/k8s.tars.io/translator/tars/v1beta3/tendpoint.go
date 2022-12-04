package v1beta3

import (
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
)

func buildTEndpoint(tserver *tarsV1beta3.TServer) *tarsV1beta3.TEndpoint {
	endpoint := &tarsV1beta3.TEndpoint{
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
	return endpoint
}

func syncTEndpoint(tserver *tarsV1beta3.TServer, endpoint *tarsV1beta3.TEndpoint) {
	endpoint.Labels = tserver.Labels
	endpoint.OwnerReferences = []k8sMetaV1.OwnerReference{
		*k8sMetaV1.NewControllerRef(tserver, tarsV1beta3.SchemeGroupVersion.WithKind(tarsMeta.TServerKind)),
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
