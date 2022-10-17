package v1beta2

import (
	"context"
	"e2e/scaffold"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1Beta2 "k8s.tars.io/crd/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"time"
)

var _ = ginkgo.Describe("try create/update normal server and check statefulset", func() {
	opts := &scaffold.Options{
		Name:      "default",
		K8SConfig: scaffold.GetK8SConfigFile(),
		SyncTime:  1500 * time.Millisecond,
	}

	s := scaffold.NewScaffold(opts)

	var Resource = "test-testserver"
	var App = "Test"
	var Server = "TestServer"

	ginkgo.BeforeEach(func() {
		tsLayout := &tarsCrdV1Beta2.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      Resource,
				Namespace: s.Namespace,
			},
			Spec: tarsCrdV1Beta2.TServerSpec{
				App:       App,
				Server:    Server,
				SubType:   tarsCrdV1Beta2.Normal,
				Important: 5,
				Normal: &tarsCrdV1Beta2.TServerNormal{
					Ports: []*tarsCrdV1Beta2.TServerPort{
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
					},
				},
				K8S: tarsCrdV1Beta2.TServerK8S{
					AbilityAffinity: tarsCrdV1Beta2.None,
					NodeSelector:    []k8sCoreV1.NodeSelectorRequirement{},
					ImagePullPolicy: k8sCoreV1.PullAlways,
					LauncherType:    tarsMeta.Background,
				},
			},
		}
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Create(context.TODO(), tsLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)
	})

	ginkgo.It("before update", func() {
		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		expectedLabels := map[string]string{
			tarsMeta.TServerAppLabel:  App,
			tarsMeta.TServerNameLabel: Server,
		}
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, statefulset.Labels))
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, statefulset.Spec.Template.Labels))
		assert.True(ginkgo.GinkgoT(), scaffold.CheckLeftInRight(expectedLabels, statefulset.Spec.Selector.MatchLabels))

		spec := &statefulset.Spec.Template.Spec
		containerPorts := spec.Containers[0].Ports
		assert.Equal(ginkgo.GinkgoT(), 2, len(containerPorts))

		var p0Name = "first"
		var p1Name = "second"
		var p0, p1 *k8sCoreV1.ContainerPort
		for i := range containerPorts {
			if containerPorts[i].Name == p0Name {
				p0 = &containerPorts[i]
				continue
			}

			if containerPorts[i].Name == p1Name {
				p1 = &containerPorts[i]
				continue
			}
			assert.True(ginkgo.GinkgoT(), false, "unexpected container port name")
		}
		assert.Equal(ginkgo.GinkgoT(), int32(10000), p0.ContainerPort)
		assert.Equal(ginkgo.GinkgoT(), int32(10001), p1.ContainerPort)
	})

	ginkgo.Context("abilityAffinity", func() {
		ginkgo.It("None", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/abilityAffinity",
					Value: tarsCrdV1Beta2.None,
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)

			spec := &statefulset.Spec.Template.Spec
			assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			expectedAffinity := &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
					{
						MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
							{
								Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
								Operator: k8sCoreV1.NodeSelectorOpExists,
							},
						},
					},
				}},
			}
			assert.Equal(ginkgo.GinkgoT(), expectedAffinity, spec.Affinity.NodeAffinity)
		})

		ginkgo.It("AppRequired", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/abilityAffinity",
					Value: tarsCrdV1Beta2.AppRequired,
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)

			spec := &statefulset.Spec.Template.Spec

			assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			expectedAffinity := &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
								{
									Key:      fmt.Sprintf("%s.%s.%s", tarsMeta.TarsAbilityLabelPrefix, s.Namespace, App),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			}
			assert.Equal(ginkgo.GinkgoT(), expectedAffinity, spec.Affinity.NodeAffinity)
		})

		ginkgo.It("ServerRequired", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/abilityAffinity",
					Value: tarsCrdV1Beta2.ServerRequired,
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)

			spec := &statefulset.Spec.Template.Spec

			assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			expectedAffinity := &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
					{
						MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
							{
								Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
								Operator: k8sCoreV1.NodeSelectorOpExists,
							},
							{
								Key:      fmt.Sprintf("%s.%s.%s-%s", tarsMeta.TarsAbilityLabelPrefix, s.Namespace, App, Server),
								Operator: k8sCoreV1.NodeSelectorOpExists,
							},
						},
					},
				}},
			}
			assert.Equal(ginkgo.GinkgoT(), expectedAffinity, spec.Affinity.NodeAffinity)
		})

		ginkgo.It("AppOrServerPreferred", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/abilityAffinity",
					Value: tarsCrdV1Beta2.AppOrServerPreferred,
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)

			spec := &statefulset.Spec.Template.Spec

			assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			expectedAffinity := &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
				PreferredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PreferredSchedulingTerm{
					{
						Weight: 60,
						Preference: k8sCoreV1.NodeSelectorTerm{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s.%s-%s", tarsMeta.TarsAbilityLabelPrefix, s.Namespace, App, Server),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
					{
						Weight: 30,
						Preference: k8sCoreV1.NodeSelectorTerm{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s.%s", tarsMeta.TarsAbilityLabelPrefix, s.Namespace, App),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			}
			assert.Equal(ginkgo.GinkgoT(), expectedAffinity, spec.Affinity.NodeAffinity)
		})
	})

	ginkgo.It("daemonSet", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchAdd,
				Path:  "/spec/k8s/daemonSet",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		_, err = s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.NotNil(ginkgo.GinkgoT(), err)
		assert.True(ginkgo.GinkgoT(), errors.IsNotFound(err))
	})

	ginkgo.It("env", func() {
		var firstEnvName = scaffold.RandStringRunes(5)
		var firstEnvValue = scaffold.RandStringRunes(64)

		var secondEnvName = scaffold.RandStringRunes(5)
		var thirdEnvName = scaffold.RandStringRunes(5)

		var keyRefOptional = true
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchAdd,
				Path: "/spec/k8s/env",
				Value: []k8sCoreV1.EnvVar{
					{
						Name:  firstEnvName,
						Value: firstEnvValue,
					},
					{
						Name: secondEnvName,
						ValueFrom: &k8sCoreV1.EnvVarSource{
							ConfigMapKeyRef: &k8sCoreV1.ConfigMapKeySelector{
								LocalObjectReference: k8sCoreV1.LocalObjectReference{
									Name: "config",
								},
								Key:      secondEnvName,
								Optional: &keyRefOptional,
							},
						},
					},
					{
						Name: thirdEnvName,
						ValueFrom: &k8sCoreV1.EnvVarSource{
							FieldRef: &k8sCoreV1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "metadata.name",
							},
						},
					},
				},
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.NotNil(ginkgo.GinkgoT(), spec.Containers[0].Env)
		expectedEnv := []k8sCoreV1.EnvVar{
			{
				Name:  firstEnvName,
				Value: firstEnvValue,
			},
			{
				Name:  secondEnvName,
				Value: "",
				ValueFrom: &k8sCoreV1.EnvVarSource{
					ConfigMapKeyRef: &k8sCoreV1.ConfigMapKeySelector{
						LocalObjectReference: k8sCoreV1.LocalObjectReference{
							Name: "config",
						},
						Key:      secondEnvName,
						Optional: &keyRefOptional,
					},
				},
			},
			{
				Name:  thirdEnvName,
				Value: "",
				ValueFrom: &k8sCoreV1.EnvVarSource{
					FieldRef: &k8sCoreV1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
		}
		assert.Equal(ginkgo.GinkgoT(), expectedEnv, spec.Containers[0].Env)
	})

	ginkgo.It("envFrom", func() {
		keyRefOptional := true
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchAdd,
				Path: "/spec/k8s/envFrom",
				Value: []k8sCoreV1.EnvFromSource{
					{
						Prefix: "",
						ConfigMapRef: &k8sCoreV1.ConfigMapEnvSource{
							LocalObjectReference: k8sCoreV1.LocalObjectReference{
								Name: "configmap",
							},
							Optional: &keyRefOptional,
						},
					},
					{
						SecretRef: &k8sCoreV1.SecretEnvSource{
							LocalObjectReference: k8sCoreV1.LocalObjectReference{
								Name: "secret",
							},
						},
					},
				},
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.NotNil(ginkgo.GinkgoT(), spec.Containers[0].EnvFrom)
		expectedEnvFrom := []k8sCoreV1.EnvFromSource{
			{
				Prefix: "",
				ConfigMapRef: &k8sCoreV1.ConfigMapEnvSource{
					LocalObjectReference: k8sCoreV1.LocalObjectReference{
						Name: "configmap",
					},
					Optional: &keyRefOptional,
				},
			},
			{
				Prefix: "",
				SecretRef: &k8sCoreV1.SecretEnvSource{
					LocalObjectReference: k8sCoreV1.LocalObjectReference{
						Name: "secret",
					},
					Optional: &keyRefOptional,
				},
			},
		}
		assert.Equal(ginkgo.GinkgoT(), expectedEnvFrom, spec.Containers[0].EnvFrom)
	})

	ginkgo.It("hostNetWork", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/k8s/hostNetwork",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.Equal(ginkgo.GinkgoT(), true, spec.HostNetwork)
	})

	ginkgo.It("hostIPC", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/k8s/hostIPC",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.Equal(ginkgo.GinkgoT(), true, spec.HostIPC)
	})

	ginkgo.It("hostPort", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchAdd,
				Path: "/spec/k8s/hostPorts",
				Value: []*tarsCrdV1Beta2.TK8SHostPort{
					{
						NameRef: "first",
						Port:    9990,
					},
					{
						NameRef: "second",
						Port:    9991,
					},
				},
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		containerPorts := spec.Containers[0].Ports
		assert.Equal(ginkgo.GinkgoT(), 2, len(containerPorts))

		var p0Name = "first"
		var p1Name = "second"
		var p0, p1 *k8sCoreV1.ContainerPort
		for i := range containerPorts {
			if containerPorts[i].Name == p0Name {
				p0 = &containerPorts[i]
				continue
			}

			if containerPorts[i].Name == p1Name {
				p1 = &containerPorts[i]
				continue
			}
			assert.True(ginkgo.GinkgoT(), false, "unexpected container port name")
		}

		assert.Equal(ginkgo.GinkgoT(), int32(10000), p0.ContainerPort)
		assert.Equal(ginkgo.GinkgoT(), int32(9990), p0.HostPort)

		assert.Equal(ginkgo.GinkgoT(), int32(10001), p1.ContainerPort)
		assert.Equal(ginkgo.GinkgoT(), int32(9991), p1.HostPort)
	})

	ginkgo.It("mounts", func() {
		hostPathType := k8sCoreV1.HostPathUnset
		quantity, _ := resource.ParseQuantity("1G")
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchAdd,
				Path: "/spec/k8s/mounts",
				Value: []tarsCrdV1Beta2.TK8SMount{
					{
						Name: "m0",
						Source: tarsCrdV1Beta2.TK8SMountSource{
							EmptyDir: &k8sCoreV1.EmptyDirVolumeSource{},
						},
						MountPath: "/empty",
					},
					{
						Name: "m1",
						Source: tarsCrdV1Beta2.TK8SMountSource{
							HostPath: &k8sCoreV1.HostPathVolumeSource{
								Path: "/host",
								Type: &hostPathType,
							},
						},
						MountPath: "/host",
					},
					{
						Name: "m2",
						Source: tarsCrdV1Beta2.TK8SMountSource{
							ConfigMap: &k8sCoreV1.ConfigMapVolumeSource{
								LocalObjectReference: k8sCoreV1.LocalObjectReference{
									Name: "configmap",
								},
							},
						},
						MountPath: "/configmap",
					},
					{
						Name: "m3",
						Source: tarsCrdV1Beta2.TK8SMountSource{
							PersistentVolumeClaim: &k8sCoreV1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc",
							},
						},
						MountPath: "/pvc",
					},
					{
						Name: "m4",
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
					{
						Name: "m5",
						Source: tarsCrdV1Beta2.TK8SMountSource{
							TLocalVolume: &tarsCrdV1Beta2.TLocalVolume{},
						},
						MountPath: "/tlv",
					},
				},
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		volumes := spec.Volumes
		assert.Equal(ginkgo.GinkgoT(), 5, len(volumes))

		for i := range volumes {
			if volumes[i].Name == "m0" {
				continue
			}

			if volumes[i].Name == "m1" {
				continue
			}

			if volumes[i].Name == "m2" {
				continue
			}

			if volumes[i].Name == "m3" {
				continue
			}

			if volumes[i].Name == "host-timezone" {
				continue
			}

			assert.True(ginkgo.GinkgoT(), false, "unexpected volumes name")
		}

		mounts := spec.Containers[0].VolumeMounts
		assert.Equal(ginkgo.GinkgoT(), 7, len(spec.Containers[0].VolumeMounts))

		for i := range mounts {
			if mounts[i].Name == "m0" {
				continue
			}
			if mounts[i].Name == "m1" {
				continue
			}
			if mounts[i].Name == "m2" {
				continue
			}
			if mounts[i].Name == "m3" {
				continue
			}
			if mounts[i].Name == "m4" {
				continue
			}
			if mounts[i].Name == "m5" {
				continue
			}
			if mounts[i].Name == "host-timezone" {
				continue
			}

			assert.True(ginkgo.GinkgoT(), false, "unexpected mounts name")
		}
	})

	ginkgo.Context("nodeSelector", func() {
		ginkgo.It("None", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:   tarsMeta.JsonPatchReplace,
					Path: "/spec/k8s/nodeSelector",
					Value: []k8sCoreV1.NodeSelectorRequirement{
						{
							Key:      tarsMeta.K8SHostNameLabel,
							Operator: k8sCoreV1.NodeSelectorOpExists,
						},
						{
							Key:      "MyLabel",
							Operator: k8sCoreV1.NodeSelectorOpIn,
							Values:   []string{"v1", "v2"},
						},
						{
							Key:      "Version",
							Operator: k8sCoreV1.NodeSelectorOpLt,
							Values:   []string{"v1"},
						},
					},
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)

			spec := &statefulset.Spec.Template.Spec

			assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			expectedAffinity := &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      tarsMeta.K8SHostNameLabel,
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
								{
									Key:      "MyLabel",
									Operator: k8sCoreV1.NodeSelectorOpIn,
									Values:   []string{"v1", "v2"},
								},
								{
									Key:      "Version",
									Operator: k8sCoreV1.NodeSelectorOpLt,
									Values:   []string{"v1"},
								},
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			}
			assert.Equal(ginkgo.GinkgoT(), expectedAffinity, spec.Affinity.NodeAffinity)
		})
	})

	ginkgo.Context("abilityAffinity & nodeSelector", func() {
		ginkgo.It("AppRequired", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/abilityAffinity",
					Value: tarsCrdV1Beta2.AppRequired,
				},
				{
					OP:   tarsMeta.JsonPatchReplace,
					Path: "/spec/k8s/nodeSelector",
					Value: []k8sCoreV1.NodeSelectorRequirement{
						{
							Key:      tarsMeta.K8SHostNameLabel,
							Operator: k8sCoreV1.NodeSelectorOpExists,
						},
						{
							Key:      "MyLabel",
							Operator: k8sCoreV1.NodeSelectorOpIn,
							Values:   []string{"v1", "v2"},
						},
						{
							Key:      "Version",
							Operator: k8sCoreV1.NodeSelectorOpLt,
							Values:   []string{"v1"},
						},
					},
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)

			spec := &statefulset.Spec.Template.Spec

			assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			expectedAffinity := &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      tarsMeta.K8SHostNameLabel,
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
								{
									Key:      "MyLabel",
									Operator: k8sCoreV1.NodeSelectorOpIn,
									Values:   []string{"v1", "v2"},
								},
								{
									Key:      "Version",
									Operator: k8sCoreV1.NodeSelectorOpLt,
									Values:   []string{"v1"},
								},
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
								{
									Key:      fmt.Sprintf("%s.%s.%s", tarsMeta.TarsAbilityLabelPrefix, s.Namespace, App),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			}
			assert.Equal(ginkgo.GinkgoT(), expectedAffinity, spec.Affinity.NodeAffinity)
		})

		ginkgo.It("AppOrServerPreferred", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/abilityAffinity",
					Value: tarsCrdV1Beta2.AppOrServerPreferred,
				},
				{
					OP:   tarsMeta.JsonPatchReplace,
					Path: "/spec/k8s/nodeSelector",
					Value: []k8sCoreV1.NodeSelectorRequirement{
						{
							Key:      tarsMeta.K8SHostNameLabel,
							Operator: k8sCoreV1.NodeSelectorOpExists,
						},
						{
							Key:      "MyLabel",
							Operator: k8sCoreV1.NodeSelectorOpIn,
							Values:   []string{"v1", "v2"},
						},
						{
							Key:      "Version",
							Operator: k8sCoreV1.NodeSelectorOpLt,
							Values:   []string{"v1"},
						},
					},
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)

			spec := &statefulset.Spec.Template.Spec

			assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			expectedAffinity := &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      tarsMeta.K8SHostNameLabel,
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
								{
									Key:      "MyLabel",
									Operator: k8sCoreV1.NodeSelectorOpIn,
									Values:   []string{"v1", "v2"},
								},
								{
									Key:      "Version",
									Operator: k8sCoreV1.NodeSelectorOpLt,
									Values:   []string{"v1"},
								},
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
				PreferredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PreferredSchedulingTerm{
					{
						Weight: 60,
						Preference: k8sCoreV1.NodeSelectorTerm{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s.%s-%s", tarsMeta.TarsAbilityLabelPrefix, s.Namespace, App, Server),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
					{
						Weight: 30,
						Preference: k8sCoreV1.NodeSelectorTerm{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s.%s", tarsMeta.TarsAbilityLabelPrefix, s.Namespace, App),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			}
			assert.Equal(ginkgo.GinkgoT(), expectedAffinity, spec.Affinity.NodeAffinity)
		})
	})

	ginkgo.It("notStacked", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/k8s/notStacked",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		expectedAffinity := &k8sCoreV1.Affinity{
			NodeAffinity: &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
			PodAntiAffinity: &k8sCoreV1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PodAffinityTerm{
					{
						LabelSelector: &k8sMetaV1.LabelSelector{
							MatchLabels: map[string]string{
								tarsMeta.TServerAppLabel:  App,
								tarsMeta.TServerNameLabel: Server,
							},
						},
						Namespaces:  []string{s.Namespace},
						TopologyKey: tarsMeta.K8SHostNameLabel,
					},
				},
			},
		}
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.NodeAffinity, spec.Affinity.NodeAffinity)
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.PodAffinity, spec.Affinity.PodAffinity)
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.PodAntiAffinity, spec.Affinity.PodAntiAffinity)
	})

	ginkgo.It("notStacked && hostIPC", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/k8s/notStacked",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		expectedAffinity := &k8sCoreV1.Affinity{
			NodeAffinity: &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
			PodAntiAffinity: &k8sCoreV1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PodAffinityTerm{
					{
						LabelSelector: &k8sMetaV1.LabelSelector{
							MatchLabels: map[string]string{
								tarsMeta.TServerAppLabel:  App,
								tarsMeta.TServerNameLabel: Server,
							},
						},
						Namespaces:  []string{s.Namespace},
						TopologyKey: tarsMeta.K8SHostNameLabel,
					},
				},
			},
		}
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.NodeAffinity, spec.Affinity.NodeAffinity)
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.PodAffinity, spec.Affinity.PodAffinity)
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.PodAntiAffinity, spec.Affinity.PodAntiAffinity)
	})

	ginkgo.It("notStacked && hostNetwork", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/k8s/notStacked",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		expectedAffinity := &k8sCoreV1.Affinity{
			NodeAffinity: &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
			PodAntiAffinity: &k8sCoreV1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PodAffinityTerm{
					{
						LabelSelector: &k8sMetaV1.LabelSelector{
							MatchLabels: map[string]string{
								tarsMeta.TServerAppLabel:  App,
								tarsMeta.TServerNameLabel: Server,
							},
						},
						Namespaces:  []string{s.Namespace},
						TopologyKey: tarsMeta.K8SHostNameLabel,
					},
				},
			},
		}
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.NodeAffinity, spec.Affinity.NodeAffinity)
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.PodAffinity, spec.Affinity.PodAffinity)
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.PodAntiAffinity, spec.Affinity.PodAntiAffinity)
	})

	ginkgo.It("notStacked && hostPort", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/k8s/notStacked",
				Value: true,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.NotNil(ginkgo.GinkgoT(), spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		expectedAffinity := &k8sCoreV1.Affinity{
			NodeAffinity: &k8sCoreV1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      fmt.Sprintf("%s.%s", tarsMeta.TarsNodeLabel, s.Namespace),
									Operator: k8sCoreV1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
			PodAntiAffinity: &k8sCoreV1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PodAffinityTerm{
					{
						LabelSelector: &k8sMetaV1.LabelSelector{
							MatchLabels: map[string]string{
								tarsMeta.TServerAppLabel:  App,
								tarsMeta.TServerNameLabel: Server,
							},
						},
						Namespaces:  []string{s.Namespace},
						TopologyKey: tarsMeta.K8SHostNameLabel,
					},
				},
			},
		}
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.NodeAffinity, spec.Affinity.NodeAffinity)
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.PodAffinity, spec.Affinity.PodAffinity)
		assert.Equal(ginkgo.GinkgoT(), expectedAffinity.PodAntiAffinity, spec.Affinity.PodAntiAffinity)
	})

	ginkgo.It("podManagementPolicy", func() {
		tserver, _ := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.NotNil(ginkgo.GinkgoT(), tserver)
		currentPolicy := tserver.Spec.K8S.PodManagementPolicy
		var targetPolicy k8sAppsV1.PodManagementPolicyType
		if currentPolicy != k8sAppsV1.ParallelPodManagement {
			targetPolicy = k8sAppsV1.ParallelPodManagement
		} else {
			targetPolicy = k8sAppsV1.OrderedReadyPodManagement
		}

		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchReplace,
				Path:  "/spec/k8s/podManagementPolicy",
				Value: targetPolicy,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		tserver, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), tserver)
		assert.Equal(ginkgo.GinkgoT(), targetPolicy, tserver.Spec.K8S.PodManagementPolicy)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)
		assert.Equal(ginkgo.GinkgoT(), currentPolicy, statefulset.Spec.PodManagementPolicy)
	})

	ginkgo.Context("readinessGate", func() {
		ginkgo.It("new readinessGate", func() {
			newReadiesGate := scaffold.RandStringRunes(10)
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchAdd,
					Path:  "/spec/k8s/readinessGate",
					Value: newReadiesGate,
				},
			}
			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)

			spec := &statefulset.Spec.Template.Spec
			assert.NotNil(ginkgo.GinkgoT(), spec.ReadinessGates)
			expectedReadinessGates := []k8sCoreV1.PodReadinessGate{
				{
					ConditionType: k8sCoreV1.PodConditionType(newReadiesGate),
				},
			}
			assert.Equal(ginkgo.GinkgoT(), expectedReadinessGates, spec.ReadinessGates)
		})
	})

	ginkgo.Context("release", func() {

		ginkgo.It("before release", func() {
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/replicas",
					Value: 3,
				},
			}

			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})

			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)
			assert.Equal(ginkgo.GinkgoT(), int32(0), *statefulset.Spec.Replicas)

			assert.Equal(ginkgo.GinkgoT(), 0, len(statefulset.Spec.Template.Spec.InitContainers))

			assert.Equal(ginkgo.GinkgoT(), 1, len(statefulset.Spec.Template.Spec.Containers))
			container := statefulset.Spec.Template.Spec.Containers[0]
			assert.Equal(ginkgo.GinkgoT(), fmt.Sprintf("%s-%s", strings.ToLower(App), strings.ToLower(Server)), container.Name)
			assert.Equal(ginkgo.GinkgoT(), tarsMeta.ServiceImagePlaceholder, container.Image)

		})

		ginkgo.It("release", func() {

			now := k8sMetaV1.Now()
			release := tarsCrdV1Beta2.TServerRelease{
				ID:     "v1beta2",
				Image:  "www.docker.com:5050/test123:v1beta2",
				Secret: "",
				Time:   &now,
				TServerReleaseNode: &tarsCrdV1Beta2.TServerReleaseNode{
					Image:  "www.docker.com:5050/node:v1beta2",
					Secret: "tars-image-secret",
				},
			}

			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/release",
					Value: release,
				},
				{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  "/spec/k8s/replicas",
					Value: 3,
				},
			}

			bs, _ := json.Marshal(jsonPatch)
			_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(s.Opts.SyncTime)

			statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})

			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), statefulset)
			assert.Equal(ginkgo.GinkgoT(), int32(3), *statefulset.Spec.Replicas)

			assert.Equal(ginkgo.GinkgoT(), 0, len(statefulset.Spec.Template.Spec.InitContainers))

			assert.Equal(ginkgo.GinkgoT(), 1, len(statefulset.Spec.Template.Spec.Containers))
			container := statefulset.Spec.Template.Spec.Containers[0]
			assert.Equal(ginkgo.GinkgoT(), fmt.Sprintf("%s-%s", strings.ToLower(App), strings.ToLower(Server)), container.Name)
			assert.Equal(ginkgo.GinkgoT(), release.Image, container.Image)

			if release.Secret != "" {
				expectedSecrets := []k8sCoreV1.LocalObjectReference{
					{
						Name: release.Secret,
					},
				}
				assert.Equal(ginkgo.GinkgoT(), expectedSecrets, statefulset.Spec.Template.Spec.ImagePullSecrets)
			}
		})
	})

	ginkgo.It("serviceAccount", func() {
		newServiceAccount := scaffold.RandStringRunes(15)
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:    tarsMeta.JsonPatchAdd,
				Path:  "/spec/k8s/serviceAccount",
				Value: newServiceAccount,
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		assert.Equal(ginkgo.GinkgoT(), newServiceAccount, spec.ServiceAccountName)
	})

	ginkgo.It("resources", func() {
		jsonPatch := tarsMeta.JsonPatch{
			{
				OP:   tarsMeta.JsonPatchAdd,
				Path: "/spec/k8s/resources",
				Value: k8sCoreV1.ResourceRequirements{
					Limits: k8sCoreV1.ResourceList{
						k8sCoreV1.ResourceCPU:    resource.MustParse("120"),
						k8sCoreV1.ResourceMemory: resource.MustParse("2000M"),
					},
					Requests: k8sCoreV1.ResourceList{
						k8sCoreV1.ResourceCPU:    resource.MustParse("100"),
						k8sCoreV1.ResourceMemory: resource.MustParse("1000M"),
					},
				},
			},
		}
		bs, _ := json.Marshal(jsonPatch)
		_, err := s.CRDClient.CrdV1beta2().TServers(s.Namespace).Patch(context.TODO(), Resource, patchTypes.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(s.Opts.SyncTime)

		statefulset, err := s.K8SClient.AppsV1().StatefulSets(s.Namespace).Get(context.TODO(), Resource, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), statefulset)

		spec := &statefulset.Spec.Template.Spec

		cpuLimit, ok := spec.Containers[0].Resources.Limits[k8sCoreV1.ResourceCPU]
		assert.True(ginkgo.GinkgoT(), ok)
		assert.NotNil(ginkgo.GinkgoT(), cpuLimit)
		assert.Equal(ginkgo.GinkgoT(), 0, cpuLimit.Cmp(resource.MustParse("120")))

		memoryLimit, ok := spec.Containers[0].Resources.Limits[k8sCoreV1.ResourceMemory]
		assert.True(ginkgo.GinkgoT(), ok)
		assert.NotNil(ginkgo.GinkgoT(), memoryLimit)
		assert.Equal(ginkgo.GinkgoT(), 0, memoryLimit.Cmp(resource.MustParse("2000M")))

		cpuRequest, ok := spec.Containers[0].Resources.Requests[k8sCoreV1.ResourceCPU]
		assert.True(ginkgo.GinkgoT(), ok)
		assert.NotNil(ginkgo.GinkgoT(), cpuRequest)
		assert.Equal(ginkgo.GinkgoT(), 0, cpuRequest.Cmp(resource.MustParse("100")))

		memoryRequest, ok := spec.Containers[0].Resources.Requests[k8sCoreV1.ResourceMemory]
		assert.True(ginkgo.GinkgoT(), ok)
		assert.NotNil(ginkgo.GinkgoT(), memoryRequest)
		assert.Equal(ginkgo.GinkgoT(), 0, memoryRequest.Cmp(resource.MustParse("1000M")))
	})
})
