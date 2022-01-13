package v1beta2

import (
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	"testing"
)

type TestCase struct {
	TS        *crdV1beta2.TServer
	SS        *k8sCoreV1.Service
	STS       *k8sAppsV1.StatefulSet
	DAEMONSET *k8sAppsV1.DaemonSet
}

var testCases = []TestCase{
	{
		TS: &crdV1beta2.TServer{
			TypeMeta: k8sMetaV1.TypeMeta{},
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "tars-tarsregistry",
				Namespace: "",
			},
			Spec: crdV1beta2.TServerSpec{
				App:       "tars",
				Server:    "tarsregistry",
				SubType:   "normal",
				Important: 5,
				Tars:       nil,
				Normal: &crdV1beta2.TServerNormal{
					Ports: []*crdV1beta2.TServerPort{
						{
							Name: "queryobj",
							Port: 17890,
						},
						{
							Name: "registryobj",
							Port: 17891,
						},
					},
				},
				K8S: crdV1beta2.TServerK8S{
					ServiceAccount:      "tarsregistry",
					Env:                 nil,
					EnvFrom:             nil,
					HostIPC:             false,
					HostNetwork:         false,
					HostPorts:           nil,
					Mounts:              nil,
					DaemonSet:           false,
					NodeSelector:        nil,
					AbilityAffinity:     "",
					PodManagementPolicy: "",
					ReadinessGate:       nil,
					Resources:           k8sCoreV1.ResourceRequirements{},
					UpdateStrategy:      k8sAppsV1.StatefulSetUpdateStrategy{},
					ImagePullPolicy:     "",
					LauncherType:        "",
				},
				Release: &crdV1beta2.TServerRelease{
					ID:    "",
					Image: "",
					Time:  nil,
				},
			},
		},
	},
}

func TestBuildStatefulsetVolumeClaimTemplates(t *testing.T) {
}

func TestBuildTVolumeClaimTemplates(t *testing.T) {
}

func TestBuildService(t *testing.T) {
	/* case1 */
	{
		caseName := "case1"
		ts := &crdV1beta2.TServer{
			TypeMeta: k8sMetaV1.TypeMeta{},
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "tars-tarsregistry",
				Namespace: "tars",
			},
			Spec: crdV1beta2.TServerSpec{
				App:       "tars",
				Server:    "tarsregistry",
				SubType:   "normal",
				Important: 5,
				Normal: &crdV1beta2.TServerNormal{
					Ports: []*crdV1beta2.TServerPort{
						{
							Name:  "queryobj",
							Port:  17890,
							IsTcp: true,
						},
						{
							Name:  "registryobj",
							Port:  17891,
							IsTcp: true,
						},
					},
				},
			},
		}
		expectedSS := &k8sCoreV1.Service{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      ts.Name,
				Namespace: ts.Namespace,
				Labels: map[string]string{
					crdMeta.TServerAppLabel:  ts.Spec.App,
					crdMeta.TServerNameLabel: ts.Spec.Server,
				},
				OwnerReferences: []k8sMetaV1.OwnerReference{
					*k8sMetaV1.NewControllerRef(ts, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
				},
			},
			Spec: k8sCoreV1.ServiceSpec{
				Selector: map[string]string{
					crdMeta.TServerAppLabel:  "tars",
					crdMeta.TServerNameLabel: "tarsregistry",
				},
				Ports: []k8sCoreV1.ServicePort{
					{
						Name:       "queryobj",
						Protocol:   k8sCoreV1.ProtocolTCP,
						Port:       17890,
						TargetPort: intstr.FromInt(17890),
					},
					{
						Name:       "registryobj",
						Protocol:   k8sCoreV1.ProtocolTCP,
						Port:       17891,
						TargetPort: intstr.FromInt(17891),
					},
				},
				ClusterIP: k8sCoreV1.ClusterIPNone,
				Type:      k8sCoreV1.ServiceTypeClusterIP,
			},
			Status: k8sCoreV1.ServiceStatus{},
		}
		ss := buildService(ts)
		if !equality.Semantic.DeepEqual(ss, expectedSS) {
			t.Errorf("failed case %s", caseName)
		}
	}
	/* case 1 */

	/* case 2 */
	{
		//caseName := "case2"
		//ts := &crdV1beta2.TServer{
		//	TypeMeta: k8sMetaV1.TypeMeta{},
		//	ObjectMeta: k8sMetaV1.ObjectMeta{
		//		Name:      "tars-tarsconfig",
		//		Namespace: "tars",
		//	},
		//	Spec: crdV1beta2.TServerSpec{
		//		App:       "tars",
		//		Server:    "tarsconfig",
		//		SubType:   "tars",
		//		Important: 5,
		//		Tars: &crdV1beta2.TServerTars{
		//			Template:    "tars.cpp",
		//			Profile:     "",
		//			AsyncThread: 5,
		//			Servants: []*crdV1beta2.TServerServant{
		//				{
		//					Name:       "ConfigObj",
		//					Port:       11111,
		//					Thread:     3,
		//					Connection: 10000,
		//					Capacity:   10000,
		//					Timeout:    6000,
		//					IsTars:      true,
		//					IsTcp:      true,
		//				},
		//			},
		//		},
		//	},
		//}
		//expectedSS := &k8sCoreV1.Service{
		//	ObjectMeta: k8sMetaV1.ObjectMeta{
		//		Name:      ts.Name,
		//		Namespace: ts.Namespace,
		//		Labels: map[string]string{
		//			crdMeta.TServerAppLabel:  ts.Spec.App,
		//			crdMeta.TServerNameLabel: ts.Spec.Server,
		//		},
		//		OwnerReferences: []k8sMetaV1.OwnerReference{
		//			*k8sMetaV1.NewControllerRef(ts, crdV1beta2.SchemeGroupVersion.WithKind(crdMeta.TServerKind)),
		//		},
		//	},
		//	Spec: k8sCoreV1.ServiceSpec{
		//		Selector: map[string]string{
		//			crdMeta.TServerAppLabel:  "tars",
		//			crdMeta.TServerNameLabel: "tarsconfig",
		//		},
		//		Ports: []k8sCoreV1.ServicePort{
		//			{
		//				Name:       "configObj",
		//				Protocol:   k8sCoreV1.ProtocolTCP,
		//				Port:       11111,
		//				TargetPort: intstr.FromInt(11111),
		//			},
		//		},
		//		ClusterIP: k8sCoreV1.ClusterIPNone,
		//		Type:      k8sCoreV1.ServiceTypeClusterIP,
		//	},
		//	Status: k8sCoreV1.ServiceStatus{},
		//}
		//ss := buildService(ts)
		//if !equality.Semantic.DeepEqual(ss, expectedSS) {
		//	t.Errorf("failed case %s", caseName)
		//}
	}

	/* case 2 */

	/* case 3 */
	{
	}
	/* case 3 */

	/* case 4 */
	{
	}
	/* case 4 */
}

func TestBuildStatefulset(t *testing.T) {
	ts := &crdV1beta2.TServer{}

	service := buildService(ts)

	if service.Name != "" {
		t.Errorf("")
	}
}

func TestBuildDaemonset(t *testing.T) {
}

func TestBuildTEndpoint(t *testing.T) {
}

func TestBuildTExitedRecord(t *testing.T) {
}
