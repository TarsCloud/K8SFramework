package v1beta2

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsCrdV1Beta2 "k8s.tars.io/crd/v1beta2"
	tarsMetaV1Beta2 "k8s.tars.io/meta/v1beta2"
	"strings"
	"time"
)

var _ = ginkgo.Describe("try create tars server with unexpected filed value", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  1500 * time.Millisecond,
	}

	s := scaffold.NewScaffold(opts)

	var tsLayout *tarsCrdV1Beta2.TServer
	ginkgo.BeforeEach(func() {
		ttLayout := &tarsCrdV1Beta2.TTemplate{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "tt.cpp",
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta2.TTemplateSpec{
				Content: "tt.cpp content",
				Parent:  "tt.cpp",
			},
		}
		_, err := s.CRDClient.CrdV1beta2().TTemplates(s.Namespace).Create(context.TODO(), ttLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		tsLayout = &tarsCrdV1Beta2.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta2.TServerSpec{
				App:       "Test",
				Server:    "TestServer",
				SubType:   "tars",
				Important: 5,
				Tars: &tarsCrdV1Beta2.TServerTars{
					Template:    "tt.cpp",
					Profile:     "",
					AsyncThread: 3,
					Servants:    []*tarsCrdV1Beta2.TServerServant{},
					Ports:       []*tarsCrdV1Beta2.TServerPort{},
				},
				Normal: nil,
				K8S: tarsCrdV1Beta2.TServerK8S{
					AbilityAffinity: tarsCrdV1Beta2.None,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					ImagePullPolicy: k8sCoreV1.PullAlways,
					LauncherType:    tarsCrdV1Beta2.Background,
				},
			},
		}
	})

	ginkgo.It("app", func() {
		tsLayout.Spec.App = "TestApp"
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
	})

	ginkgo.It("server", func() {
		tsLayout.Spec.App = "MTestServer"
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
	})

	ginkgo.It("tars.template", func() {
		tsLayout.Spec.Tars.Template = "xxx"
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
	})

	ginkgo.Context("tars.servant", func() {
		ginkgo.BeforeEach(func() {
			tsLayout.Spec.Tars.Servants = []*tarsCrdV1Beta2.TServerServant{
				{
					Thread:     3,
					Connection: 1000,
					Capacity:   1000,
					Timeout:    1000,
					IsTars:     true,
					IsTcp:      true,
				},
				{
					Thread:     3,
					Connection: 1000,
					Capacity:   1000,
					Timeout:    1000,
					IsTars:     true,
					IsTcp:      true,
				},
			}
		})

		ginkgo.It("reserved servant port", func() {
			tsLayout.Spec.Tars.Servants[0].Name = "FirstObj"
			tsLayout.Spec.Tars.Servants[0].Port = tarsMetaV1Beta2.NodeServantPort

			tsLayout.Spec.Tars.Servants[1].Name = "SecondObj"
			tsLayout.Spec.Tars.Servants[1].Port = 3000

			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate servant name", func() {
			tsLayout.Spec.Tars.Servants[0].Name = "FirstObj"
			tsLayout.Spec.Tars.Servants[0].Port = 3000

			tsLayout.Spec.Tars.Servants[1].Name = "firstObj"
			tsLayout.Spec.Tars.Servants[1].Port = 3001

			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate servant port", func() {
			tsLayout.Spec.Tars.Servants[0].Name = "FirstObj"
			tsLayout.Spec.Tars.Servants[0].Port = 3000

			tsLayout.Spec.Tars.Servants[1].Name = "SecondObj"
			tsLayout.Spec.Tars.Servants[1].Port = 3000

			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})
	})

	ginkgo.Context("tars.ports", func() {
		ginkgo.BeforeEach(func() {
			tsLayout.Spec.Tars.Ports = []*tarsCrdV1Beta2.TServerPort{
				{
					IsTcp: true,
				},
				{
					IsTcp: true,
				},
			}
		})

		ginkgo.It("reserved port value", func() {
			tsLayout.Spec.Tars.Ports[0].Name = "first"
			tsLayout.Spec.Tars.Ports[0].Port = tarsMetaV1Beta2.NodeServantPort

			tsLayout.Spec.Tars.Ports[1].Name = "second"
			tsLayout.Spec.Tars.Ports[1].Port = 3001

			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate port name", func() {
			tsLayout.Spec.Tars.Ports[0].Name = "first"
			tsLayout.Spec.Tars.Ports[0].Port = 3000

			tsLayout.Spec.Tars.Ports[1].Name = "first"
			tsLayout.Spec.Tars.Ports[1].Port = 3001

			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate port value", func() {
			tsLayout.Spec.Tars.Ports[0].Name = "first"
			tsLayout.Spec.Tars.Ports[0].Port = 3000

			tsLayout.Spec.Tars.Ports[1].Name = "second"
			tsLayout.Spec.Tars.Ports[1].Port = 3000

			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})
	})

	ginkgo.Context("tars.servant && tars.ports", func() {
		ginkgo.BeforeEach(func() {
			tsLayout.Spec.Tars.Servants = []*tarsCrdV1Beta2.TServerServant{
				{
					Thread:     3,
					Connection: 1000,
					Capacity:   1000,
					Timeout:    1000,
					IsTars:     true,
					IsTcp:      true,
				},
			}
			tsLayout.Spec.Tars.Ports = []*tarsCrdV1Beta2.TServerPort{
				{
					IsTcp: true,
				},
			}
		})
		ginkgo.It("duplicate name", func() {
			tsLayout.Spec.Tars.Servants[0].Name = "FirstObj"
			tsLayout.Spec.Tars.Servants[0].Port = 3000

			tsLayout.Spec.Tars.Ports[0].Name = "firstobj"
			tsLayout.Spec.Tars.Ports[0].Port = 30001

			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate port", func() {
			tsLayout.Spec.Tars.Servants[0].Name = "FirstObj"
			tsLayout.Spec.Tars.Servants[0].Port = 3000

			tsLayout.Spec.Tars.Ports[0].Name = "secondobj"
			tsLayout.Spec.Tars.Ports[0].Port = 3000

			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})
	})

	ginkgo.Context("k8s", func() {
		ginkgo.BeforeEach(func() {
			tsLayout.Spec.Tars.Servants = []*tarsCrdV1Beta2.TServerServant{
				{
					Name:       "FirstObj",
					Port:       10000,
					Thread:     3,
					Connection: 1000,
					Capacity:   1000,
					Timeout:    1000,
					IsTars:     true,
					IsTcp:      true,
				},
				{
					Name:       "SecondObj",
					Port:       10001,
					Thread:     3,
					Connection: 1000,
					Capacity:   1000,
					Timeout:    1000,
					IsTars:     true,
					IsTcp:      true,
				},
			}
		})

		ginkgo.It("hostPort.nameRef not exist", func() {
			tsLayout.Spec.K8S.HostPorts = []*tarsCrdV1Beta2.TK8SHostPort{
				{
					NameRef: "xxxx",
					Port:    99,
				},
			}
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate hostPorts.nameRef", func() {
			tsLayout.Spec.K8S.HostPorts = []*tarsCrdV1Beta2.TK8SHostPort{
				{
					NameRef: "FirstObj",
					Port:    99,
				},
				{
					NameRef: "FirstObj",
					Port:    100,
				},
			}
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate hostPorts.port", func() {
			tsLayout.Spec.K8S.HostPorts = []*tarsCrdV1Beta2.TK8SHostPort{
				{
					NameRef: "FirstObj",
					Port:    99,
				},
				{
					NameRef: "SecondObj",
					Port:    99,
				},
			}
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate mounts name", func() {
			var hostPathType k8sCoreV1.HostPathType = ""
			tsLayout.Spec.K8S.Mounts = []tarsCrdV1Beta2.TK8SMount{
				{
					Name: "m1",
					Source: tarsCrdV1Beta2.TK8SMountSource{
						HostPath: &k8sCoreV1.HostPathVolumeSource{
							Path: "/data",
							Type: &hostPathType,
						},
					},
					MountPath: "/data/1",
				},
				{
					Name: "m1",
					Source: tarsCrdV1Beta2.TK8SMountSource{
						EmptyDir: &k8sCoreV1.EmptyDirVolumeSource{},
					},
					MountPath: "/data/2",
				},
			}
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("daemonset and tlv", func() {
			tsLayout.Spec.K8S.DaemonSet = true
			tsLayout.Spec.K8S.Mounts = []tarsCrdV1Beta2.TK8SMount{
				{
					Name: "tlv",
					Source: tarsCrdV1Beta2.TK8SMountSource{
						TLocalVolume: &tarsCrdV1Beta2.TLocalVolume{},
					},
					MountPath: "/tlv",
				},
			}
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("daemonset and pvct", func() {
			tsLayout.Spec.K8S.DaemonSet = true
			quantity, _ := resource.ParseQuantity("1G")
			tsLayout.Spec.K8S.Mounts = []tarsCrdV1Beta2.TK8SMount{
				{
					Name: "tlv",
					Source: tarsCrdV1Beta2.TK8SMountSource{
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
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})
	})
})
