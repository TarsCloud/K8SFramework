package v1beta2

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/integer"
	tarsV1beta2 "k8s.tars.io/apis/tars/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"math"
	"regexp"
	"strconv"
	"tarswebhook/webhook/lister"
	"tarswebhook/webhook/mutating"
)

func mutatingCreateTServer(listers *lister.Listers, requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tserver := &tarsV1beta2.TServer{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tserver)

	var jsonPatch tarsTool.JsonPatch

	if tserver.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tserver.Spec.App,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tserver.Spec.Server,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1SubType",
		Value: string(tserver.Spec.SubType),
	})

	if tserver.Spec.Tars != nil {
		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Template",
			Value: tserver.Spec.Tars.Template,
		})

		if tserver.Spec.K8S.ReadinessGate != tarsMeta.TPodReadinessGate {
			jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
				OP:    tarsTool.JsonPatchAdd,
				Path:  "/spec/k8s/readinessGate",
				Value: tarsMeta.TPodReadinessGate,
			})
		}
	}

	if tserver.Spec.Normal != nil {
		if _, ok := tserver.Labels[tarsMeta.TTemplateLabel]; ok {
			jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
				OP:   tarsTool.JsonPatchRemove,
				Path: "/metadata/labels/tars.io~1Template",
			})
		}

		if tserver.Spec.Normal.Ports == nil {
			jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
				OP:    tarsTool.JsonPatchAdd,
				Path:  "/spec/normal/ports",
				Value: make([]tarsV1beta2.TServerPort, 0),
			})
		}
	}

	if len(tserver.Spec.K8S.HostPorts) > 0 || tserver.Spec.K8S.HostIPC {
		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/spec/k8s/notStacked",
			Value: true,
		})
	}

	maxReplicasValue := math.MaxInt32
	minReplicasValue := math.MinInt32

	if tserver.Annotations != nil {
		const pattern = "^[1-9]?[0-9]$"
		if maxReplicas, ok := tserver.Annotations[tarsMeta.TMaxReplicasAnnotation]; ok {
			matched, _ := regexp.MatchString(pattern, maxReplicas)
			if !matched {
				return nil, fmt.Errorf(tarsMeta.ResourceInvalidError, "tserver", "unexpected annotation format")
			}
			maxReplicasValue, _ = strconv.Atoi(maxReplicas)
		}

		if minReplicas, ok := tserver.Annotations[tarsMeta.TMinReplicasAnnotation]; ok {
			matched, _ := regexp.MatchString(pattern, minReplicas)
			if !matched {
				return nil, fmt.Errorf(tarsMeta.ResourceInvalidError, "tserver", "unexpected annotation format")
			}
			minReplicasValue, _ = strconv.Atoi(minReplicas)
		}

		if minReplicasValue > maxReplicasValue {
			return nil, fmt.Errorf(tarsMeta.ResourceInvalidError, "tserver", "unexpected annotation value")
		}

		if tserver.Spec.Release == nil {
			autoRelate, ok := tserver.Annotations[tarsMeta.TAutoReleaseAnnotation]
			if ok && autoRelate != "false" {
				appRequired, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tserver.Spec.App})
				var targetServerName string
				if tserver.Spec.App != "DCache" {
					targetServerName = tserver.Spec.Server
				} else {
					targetServerName = func(sn string) string {
						re, _ := regexp.Compile(`^.*(ProxyServer|RouterServer|TCacheServer|MKVCacheServer|DBAccessServer)(\d+-\d+)?`)
						ms := re.FindStringSubmatch(sn)
						if len(ms) >= 2 {
							return ms[1]
						}
						return sn
					}(tserver.Spec.Server)
				}

				serverRequired, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{targetServerName})
				selector := labels.NewSelector().Add(*appRequired).Add(*serverRequired)
				tis, _ := listers.TILister.TImages(tserver.Namespace).List(selector)
				for _, ti := range tis {
					if ti.Default != nil {
						id := *ti.Default
						for _, r := range ti.Releases {
							if r.ID == id {
								now := k8sMetaV1.Now()
								release := &tarsV1beta2.TServerRelease{
									ID:                 id,
									Image:              r.Image,
									Secret:             r.Secret,
									Time:               &now,
									TServerReleaseNode: nil,
								}
								tserver.Spec.Release = release
								jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
									OP:    tarsTool.JsonPatchAdd,
									Path:  "/spec/release",
									Value: release,
								})
								goto autoReleaseExit
							}
						}
					}
				}
			autoReleaseExit:
			}
		}
	}

	if tserver.Spec.Release == nil {
		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: 0,
		})
	} else {
		replicas := int(tserver.Spec.K8S.Replicas)
		replicas = integer.IntMax(replicas, minReplicasValue)
		replicas = integer.IntMin(replicas, maxReplicasValue)

		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: replicas,
		})

		if tserver.Spec.Release.Time.IsZero() {
			now := k8sMetaV1.Now()
			jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
				OP:    tarsTool.JsonPatchAdd,
				Path:  "/spec/release/time",
				Value: now.ToUnstructured(),
			})
		}

		if tserver.Spec.Tars != nil {
			if tserver.Spec.Release.TServerReleaseNode == nil || tserver.Spec.Release.TServerReleaseNode.Image == "" {
				image, secret := tarsRuntime.TFCConfig.GetDefaultNodeImage(tserver.Namespace)
				if image == tarsMeta.ServiceImagePlaceholder {
					return nil, fmt.Errorf(tarsMeta.ResourceInvalidError, tserver, "no default node image has been set")
				}

				jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
					OP:    tarsTool.JsonPatchAdd,
					Path:  "/spec/release/nodeImage",
					Value: image,
				})

				jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
					OP:    tarsTool.JsonPatchAdd,
					Path:  "/spec/release/nodeSecret",
					Value: secret,
				})
			}
		}

		if tserver.Spec.Normal != nil {
			if tserver.Spec.Release.TServerReleaseNode != nil {
				if tserver.Spec.Release.TServerReleaseNode.Image != "" {
					jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
						OP:   tarsTool.JsonPatchRemove,
						Path: "/spec/release/nodeImage",
					})
				}

				if tserver.Spec.Release.TServerReleaseNode.Secret != "" {
					jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
						OP:   tarsTool.JsonPatchRemove,
						Path: "/spec/release/nodeSecret",
					})
				}
			}
		}
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTServer(listers *lister.Listers, requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTServer(listers, requestAdmissionView)
}

func init() {
	gvr := tarsV1beta2.SchemeGroupVersion.WithResource("tservers")
	mutating.Registry(k8sAdmissionV1.Create, &gvr, mutatingCreateTServer)
	mutating.Registry(k8sAdmissionV1.Update, &gvr, mutatingUpdateTServer)
}
