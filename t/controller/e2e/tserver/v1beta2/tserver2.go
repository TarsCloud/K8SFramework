package v1beta2

import (
	"context"
	"e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsV1Beta2 "k8s.tars.io/apis/tars/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"

	"strings"
	"time"
)

var _ = ginkgo.Describe("try create normal server with unexpected filed value", func() {
	opts := &scaffold.Options{
		Name:     "default",
		SyncTime: 800 * time.Millisecond,
	}

	s := scaffold.NewScaffold(opts)

	var Resource = "test-testserver"

	var tsLayout *tarsV1Beta2.TServer
	ginkgo.BeforeEach(func() {
		tsLayout = &tarsV1Beta2.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      "test-testserver",
				Namespace: s.Namespace,
			},
			Spec: tarsV1Beta2.TServerSpec{
				App:       "Test",
				Server:    "TestServer",
				SubType:   "normal",
				Important: 5,
				Normal:    &tarsV1Beta2.TServerNormal{},
				K8S: tarsV1Beta2.TServerK8S{
					AbilityAffinity: tarsV1Beta2.None,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					ImagePullPolicy: k8sCoreV1.PullAlways,
					LauncherType:    tarsMeta.Background,
				},
			},
		}
	})

	ginkgo.AfterEach(func() {
		_ = tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Delete(context.TODO(), Resource, k8sMetaV1.DeleteOptions{})
	})

	ginkgo.It("app", func() {
		tsLayout.Spec.App = "TestApp"
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
	})

	ginkgo.It("server", func() {
		tsLayout.Spec.App = "MTestServer"
		_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
	})

	ginkgo.Context("tars.ports", func() {
		ginkgo.BeforeEach(func() {
			tsLayout.Spec.Normal.Ports = []*tarsV1Beta2.TServerPort{
				{
					IsTcp: true,
				},
				{
					IsTcp: true,
				},
			}
		})

		ginkgo.It("reserved port value", func() {
			tsLayout.Spec.Normal.Ports[0].Name = "first"
			tsLayout.Spec.Normal.Ports[0].Port = tarsMeta.NodeServantPort

			tsLayout.Spec.Normal.Ports[1].Name = "second"
			tsLayout.Spec.Normal.Ports[1].Port = 3001

			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
		})

		ginkgo.It("duplicate port name", func() {
			tsLayout.Spec.Normal.Ports[0].Name = "first"
			tsLayout.Spec.Normal.Ports[0].Port = 3000

			tsLayout.Spec.Normal.Ports[1].Name = "first"
			tsLayout.Spec.Normal.Ports[1].Port = 3001

			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate port value", func() {
			tsLayout.Spec.Normal.Ports[0].Name = "first"
			tsLayout.Spec.Normal.Ports[0].Port = 3000

			tsLayout.Spec.Normal.Ports[1].Name = "second"
			tsLayout.Spec.Normal.Ports[1].Port = 3000

			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})
	})

	ginkgo.Context("k8s", func() {
		ginkgo.BeforeEach(func() {
			tsLayout.Spec.Normal.Ports = []*tarsV1Beta2.TServerPort{
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
			tsLayout.Spec.K8S.HostPorts = []*tarsV1Beta2.TK8SHostPort{
				{
					NameRef: "xxxx",
					Port:    99,
				},
			}
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate hostPorts.nameRef", func() {
			tsLayout.Spec.K8S.HostPorts = []*tarsV1Beta2.TK8SHostPort{
				{
					NameRef: "first",
					Port:    99,
				},
				{
					NameRef: "first",
					Port:    100,
				},
			}
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate hostPorts.port", func() {
			tsLayout.Spec.K8S.HostPorts = []*tarsV1Beta2.TK8SHostPort{
				{
					NameRef: "first",
					Port:    99,
				},
				{
					NameRef: "second",
					Port:    99,
				},
			}
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("duplicate mounts name", func() {
			var hostPathType k8sCoreV1.HostPathType = ""
			tsLayout.Spec.K8S.Mounts = []tarsV1Beta2.TK8SMount{
				{
					Name: "m1",
					Source: tarsV1Beta2.TK8SMountSource{
						HostPath: &k8sCoreV1.HostPathVolumeSource{
							Path: "/data",
							Type: &hostPathType,
						},
					},
					MountPath: "/data/1",
				},
				{
					Name: "m1",
					Source: tarsV1Beta2.TK8SMountSource{
						EmptyDir: &k8sCoreV1.EmptyDirVolumeSource{},
					},
					MountPath: "/data/2",
				},
			}
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("daemonset and tlv", func() {
			tsLayout.Spec.K8S.DaemonSet = true
			tsLayout.Spec.K8S.Mounts = []tarsV1Beta2.TK8SMount{
				{
					Name: "tlv",
					Source: tarsV1Beta2.TK8SMountSource{
						TLocalVolume: &tarsV1Beta2.TLocalVolume{},
					},
					MountPath: "/tlv",
				},
			}
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})

		ginkgo.It("daemonset and pvct", func() {
			tsLayout.Spec.K8S.DaemonSet = true
			quantity, _ := resource.ParseQuantity("1G")
			tsLayout.Spec.K8S.Mounts = []tarsV1Beta2.TK8SMount{
				{
					Name: "tlv",
					Source: tarsV1Beta2.TK8SMountSource{
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
			_, err := tarsRuntime.Clients.CrdClient.TarsV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
			assert.NotNil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), strings.Contains(err.Error(), "denied the request:"))
		})
	})
})
