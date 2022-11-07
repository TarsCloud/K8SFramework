package v1beta3

import (
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	tarsApisV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"strings"
)

func buildServicePorts(tserver *tarsApisV1beta3.TServer) []k8sCoreV1.ServicePort {

	getProtocol := func(isTcp bool) k8sCoreV1.Protocol {
		if isTcp {
			return k8sCoreV1.ProtocolTCP
		}
		return k8sCoreV1.ProtocolUDP
	}

	var ports []k8sCoreV1.ServicePort
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
	}

	if tserver.Spec.Normal != nil {
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

func buildService(tserver *tarsApisV1beta3.TServer) *k8sCoreV1.Service {
	service := &k8sCoreV1.Service{
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
		Spec: k8sCoreV1.ServiceSpec{
			Ports: buildServicePorts(tserver),
			Selector: map[string]string{
				tarsMeta.TServerAppLabel:  tserver.Spec.App,
				tarsMeta.TServerNameLabel: tserver.Spec.Server,
			},
			ClusterIP: k8sCoreV1.ClusterIPNone,
			Type:      k8sCoreV1.ServiceTypeClusterIP,
		},
	}
	return service
}

func syncService(tserver *tarsApisV1beta3.TServer, service *k8sCoreV1.Service) {
	service.Spec.Ports = buildServicePorts(tserver)
}
