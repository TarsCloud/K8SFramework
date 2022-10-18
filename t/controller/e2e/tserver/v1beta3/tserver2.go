package v1beta3

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsCrdV1Beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"

	"strings"
	"time"
)

var _ = ginkgo.Describe("try create normal server with unexpected filed value", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  800 * time.Millisecond,
	}

	var Resource = "test-testserver"

	s := scaffold.NewScaffold(opts)

	var tsLayout *tarsCrdV1Beta3.TServer
	ginkgo.BeforeEach(func() {
		tsLayout = &tarsCrdV1Beta3.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta3.TServerSpec{
				App:       "Test",
				Server:    "TestServer",
				SubType:   "normal",
				Important: 5,
				Normal:    &tarsCrdV1Beta3.TServerNormal{},
				K8S: tarsCrdV1Beta3.TServerK8S{
					AbilityAffinity: tarsCrdV1Beta3.None,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					ImagePullPolicy: k8sCoreV1.PullAlways,
					LauncherType:    tarsMeta.Background,
				},
			},
		}
	})

	ginkgo.AfterEach(func() {
		_ = s.CRDClient.CrdV1beta3().TServers(s.Namespace).Delete(context.TODO(), Resource, k8sMetaV1.DeleteOptions{})
	})

	ginkgo.It("app", func() {
		tsLayout.Spec.App = "TestApp"
		_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
	})

	ginkgo.It("server", func() {
		tsLayout.Spec.App = "MTestServer"
		_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
	})

	ginkgo.Context("tars.ports", func() {
		ginkgo.BeforeEach(func() {
			tsLayout.Spec.Normal.Ports = []*tarsCrdV1Beta3.TServerPort{
				{
					IsTcp: true,
				},
				{
					IsTcp: true,
				},
			}
		})

		ginkgo.It("reserved port name", func() {
			tsLayout.Spec.Normal.Ports[0].Name = tarsMeta.NodeServantName
			tsLayout.Spec.Normal.Ports[0].Port = 3000

			tsLayout.Spec.Normal.Ports[1].Name = "second"
			tsLayout.Spec.Normal.Ports[1].Port = 3001

			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
		})

		ginkgo.It("reserved port value", func() {
			tsLayout.Spec.Normal.Ports[0].Name = "first"
			tsLayout.Spec.Normal.Ports[0].Port = tarsMeta.NodeServantPort

			tsLayout.Spec.Normal.Ports[1].Name = "second"
			tsLayout.Spec.Normal.Ports[1].Port = 3001

			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
		})

		ginkgo.It("duplicate port name", func() {
			tsLayout.Spec.Normal.Ports[0].Name = "first"
			tsLayout.Spec.Normal.Ports[0].Port = 3000

			tsLayout.Spec.Normal.Ports[1].Name = "first"
			tsLayout.Spec.Normal.Ports[1].Port = 3001

			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate port value", func() {
			tsLayout.Spec.Normal.Ports[0].Name = "first"
			tsLayout.Spec.Normal.Ports[0].Port = 3000

			tsLayout.Spec.Normal.Ports[1].Name = "second"
			tsLayout.Spec.Normal.Ports[1].Port = 3000

			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})
	})

	ginkgo.Context("k8s", func() {
		ginkgo.BeforeEach(func() {
			tsLayout.Spec.Normal.Ports = []*tarsCrdV1Beta3.TServerPort{
				{
					Name:  "first",
					Port:  10000,
					IsTcp: true,
				},
				{
					Name:  "second",
					Port:  10001,
					IsTcp: true,
				},
			}
		})

		ginkgo.It("hostPort.nameRef not exist", func() {
			tsLayout.Spec.K8S.HostPorts = []*tarsCrdV1Beta3.TK8SHostPort{
				{
					NameRef: "xxxx",
					Port:    99,
				},
			}
			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate hostPorts.nameRef", func() {
			tsLayout.Spec.K8S.HostPorts = []*tarsCrdV1Beta3.TK8SHostPort{
				{
					NameRef: "first",
					Port:    99,
				},
				{
					NameRef: "first",
					Port:    100,
				},
			}
			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate hostPorts.port", func() {
			tsLayout.Spec.K8S.HostPorts = []*tarsCrdV1Beta3.TK8SHostPort{
				{
					NameRef: "first",
					Port:    99,
				},
				{
					NameRef: "second",
					Port:    99,
				},
			}
			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate mounts name", func() {
			var hostPathType k8sCoreV1.HostPathType = ""
			tsLayout.Spec.K8S.Mounts = []tarsCrdV1Beta3.TK8SMount{
				{
					Name: "m1",
					Source: tarsCrdV1Beta3.TK8SMountSource{
						HostPath: &k8sCoreV1.HostPathVolumeSource{
							Path: "/data",
							Type: &hostPathType,
						},
					},
					MountPath: "/data/1",
				},
				{
					Name: "m1",
					Source: tarsCrdV1Beta3.TK8SMountSource{
						EmptyDir: &k8sCoreV1.EmptyDirVolumeSource{},
					},
					MountPath: "/data/2",
				},
			}
			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("daemonset and tlv", func() {
			tsLayout.Spec.K8S.DaemonSet = true
			tsLayout.Spec.K8S.Mounts = []tarsCrdV1Beta3.TK8SMount{
				{
					Name: "tlv",
					Source: tarsCrdV1Beta3.TK8SMountSource{
						TLocalVolume: &tarsCrdV1Beta3.TLocalVolume{},
					},
					MountPath: "/tlv",
				},
			}
			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("daemonset and pvct", func() {
			tsLayout.Spec.K8S.DaemonSet = true
			quantity, _ := resource.ParseQuantity("1G")
			tsLayout.Spec.K8S.Mounts = []tarsCrdV1Beta3.TK8SMount{
				{
					Name: "tlv",
					Source: tarsCrdV1Beta3.TK8SMountSource{
						PersistentVolumeClaimTemplate: &k8sCoreV1.PersistentVolumeClaim{
							TypeMeta: k8sMetaV1.TypeMeta{
								Kind:       "PersistentVolumeClaim",
								APIVersion: "v1",
							},
							ObjectMeta: k8sMetaV1.ObjectMeta{
								Name: "",
								Labels: map[string]string{
									"lk1": "lk2",
								},
								Annotations: map[string]string{
									"ak1": "ak2",
								},
							},
							Spec: k8sCoreV1.PersistentVolumeClaimSpec{
								AccessModes: []k8sCoreV1.PersistentVolumeAccessMode{k8sCoreV1.ReadWriteOnce},
								Selector: &k8sMetaV1.LabelSelector{
									MatchLabels: map[string]string{
										"aliyun.cloud.zone":    "3a",
										"aliyun.cloud.storage": "ssd",
									},
								},
								Resources: k8sCoreV1.ResourceRequirements{
									Requests: map[k8sCoreV1.ResourceName]resource.Quantity{
										k8sCoreV1.ResourceStorage: quantity,
									},
								},
							},
						},
					},
					MountPath: "/pvct",
				},
			}
			_, err := s.CRDClient.CrdV1beta3().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})
	})
})
