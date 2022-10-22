package v1beta3

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/integer"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"math"
	"regexp"
	"strconv"
	"tarscontroller/util"
)

func mutatingCreateTServer(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tserver := &tarsCrdV1beta3.TServer{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tserver)

	var jsonPatch tarsMeta.JsonPatch

	if tserver.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
		OP:    tarsMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tserver.Spec.App,
	})

	jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
		OP:    tarsMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tserver.Spec.Server,
	})

	jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
		OP:    tarsMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1SubType",
		Value: tserver.Spec.SubType,
	})

	if tserver.Spec.Tars != nil {
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Template",
			Value: tserver.Spec.Tars.Template,
		})

		gatesMap := map[string]interface{}{}
		gatesArray := []string{tarsMeta.TPodReadinessGate}
		for _, v := range tserver.Spec.K8S.ReadinessGates {
			if v != tarsMeta.TPodReadinessGate {
				if _, ok := gatesMap[v]; !ok {
					gatesArray = append(gatesArray, v)
					gatesMap[v] = nil
				}
			}
		}

		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/spec/k8s/readinessGates",
			Value: gatesArray,
		})
	}

	if tserver.Spec.Normal != nil {
		if _, ok := tserver.Labels[tarsMeta.TTemplateLabel]; ok {
			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:   tarsMeta.JsonPatchRemove,
				Path: "/metadata/labels/tars.io~1Template",
			})
		}

		if tserver.Spec.Normal.Ports == nil {
			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:    tarsMeta.JsonPatchAdd,
				Path:  "/spec/normal/ports",
				Value: make([]tarsCrdV1beta3.TServerPort, 0),
			})
		}

		if len(tserver.Spec.K8S.ReadinessGates) > 0 {
			gatesMap := map[string]interface{}{}
			var gatesArray []string

			for _, v := range tserver.Spec.K8S.ReadinessGates {
				if _, ok := gatesMap[v]; !ok {
					gatesArray = append(gatesArray, v)
					gatesMap[v] = nil
				}
			}

			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:    tarsMeta.JsonPatchAdd,
				Path:  "/spec/k8s/readinessGates",
				Value: gatesArray,
			})
		}
	}

	if len(tserver.Spec.K8S.HostPorts) > 0 || tserver.Spec.K8S.HostIPC {
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
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
	}

	if tserver.Spec.Release == nil {
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: 0,
		})
	} else {
		replicas := int(tserver.Spec.K8S.Replicas)
		replicas = integer.IntMax(replicas, minReplicasValue)
		replicas = integer.IntMin(replicas, maxReplicasValue)

		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: replicas,
		})

		if tserver.Spec.Release.Time.IsZero() {
			now := k8sMetaV1.Now()
			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:    tarsMeta.JsonPatchAdd,
				Path:  "/spec/release/time",
				Value: now.ToUnstructured(),
			})
		}

		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1ServerID",
			Value: tserver.Spec.Release.ID,
		})

		if tserver.Spec.Tars != nil {
			if tserver.Spec.Release.TServerReleaseNode == nil || tserver.Spec.Release.TServerReleaseNode.Image == "" {
				image, secret := util.GetDefaultNodeImage(tserver.Namespace)
				if image == tarsMeta.ServiceImagePlaceholder {
					return nil, fmt.Errorf(tarsMeta.ResourceInvalidError, tserver, "no default node image has been set")
				}

				jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
					OP:    tarsMeta.JsonPatchAdd,
					Path:  "/spec/release/nodeImage",
					Value: image,
				})

				jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
					OP:    tarsMeta.JsonPatchAdd,
					Path:  "/spec/release/nodeSecret",
					Value: secret,
				})
			}
		}

		if tserver.Spec.Normal != nil {
			if tserver.Spec.Release.TServerReleaseNode != nil {
				if tserver.Spec.Release.TServerReleaseNode.Image != "" {
					jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
						OP:   tarsMeta.JsonPatchRemove,
						Path: "/spec/release/nodeImage",
					})
				}

				if tserver.Spec.Release.TServerReleaseNode.Secret != "" {
					jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
						OP:   tarsMeta.JsonPatchRemove,
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

func mutatingUpdateTServer(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTServer(requestAdmissionView)
}
