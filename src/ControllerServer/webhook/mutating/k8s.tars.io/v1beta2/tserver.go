package v1beta2

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/integer"
	tarsCrdV1beta2 "k8s.tars.io/crd/v1beta2"
	tarsMetaTools "k8s.tars.io/meta/tools"
	tarsMetaV1beta2 "k8s.tars.io/meta/v1beta2"
	"math"
	"regexp"
	"strconv"
	"tarscontroller/controller"
)

func mutatingCreateTServer(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tserver := &tarsCrdV1beta2.TServer{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tserver)

	var jsonPatch tarsMetaTools.JsonPatch

	if tserver.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tserver.Spec.App,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tserver.Spec.Server,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1SubType",
		Value: string(tserver.Spec.SubType),
	})

	if tserver.Spec.Tars != nil {
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Template",
			Value: tserver.Spec.Tars.Template,
		})

		if tserver.Spec.K8S.ReadinessGate != tarsMetaV1beta2.TPodReadinessGate {
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/spec/k8s/readinessGate",
				Value: tarsMetaV1beta2.TPodReadinessGate,
			})
		}
	}

	if tserver.Spec.Normal != nil {
		if _, ok := tserver.Labels[tarsMetaV1beta2.TemplateLabel]; ok {
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:   tarsMetaTools.JsonPatchRemove,
				Path: "/metadata/labels/tars.io~1Template",
			})
		}

		if tserver.Spec.Normal.Ports == nil {
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/spec/normal/ports",
				Value: make([]tarsCrdV1beta2.TServerPort, 0),
			})
		}
	}

	if len(tserver.Spec.K8S.HostPorts) > 0 || tserver.Spec.K8S.HostIPC {
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/spec/k8s/notStacked",
			Value: true,
		})
	}

	maxReplicasValue := math.MaxInt32
	minReplicasValue := math.MinInt32

	if tserver.Annotations != nil {
		const pattern = "^[1-9]?[0-9]$"
		if maxReplicas, ok := tserver.Annotations[tarsMetaV1beta2.TMaxReplicasAnnotation]; ok {
			matched, _ := regexp.MatchString(pattern, maxReplicas)
			if !matched {
				return nil, fmt.Errorf(tarsMetaV1beta2.ResourceInvalidError, "tserver", "unexpected annotation format")
			}
			maxReplicasValue, _ = strconv.Atoi(maxReplicas)
		}

		if minReplicas, ok := tserver.Annotations[tarsMetaV1beta2.TMinReplicasAnnotation]; ok {
			matched, _ := regexp.MatchString(pattern, minReplicas)
			if !matched {
				return nil, fmt.Errorf(tarsMetaV1beta2.ResourceInvalidError, "tserver", "unexpected annotation format")
			}
			minReplicasValue, _ = strconv.Atoi(minReplicas)
		}

		if minReplicasValue > maxReplicasValue {
			return nil, fmt.Errorf(tarsMetaV1beta2.ResourceInvalidError, "tserver", "unexpected annotation value")
		}
	}

	if tserver.Spec.Release == nil {
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: 0,
		})
	} else {
		replicas := int(tserver.Spec.K8S.Replicas)
		replicas = integer.IntMax(replicas, minReplicasValue)
		replicas = integer.IntMin(replicas, maxReplicasValue)

		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: replicas,
		})

		if tserver.Spec.Release.Time.IsZero() {
			now := k8sMetaV1.Now()
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/spec/release/time",
				Value: now.ToUnstructured(),
			})
		}

		if tserver.Spec.Tars != nil {
			if tserver.Spec.Release.TServerReleaseNode == nil || tserver.Spec.Release.TServerReleaseNode.Image == "" {
				image, secret := controller.GetDefaultNodeImage(tserver.Namespace)
				if image == tarsMetaV1beta2.ServiceImagePlaceholder {
					return nil, fmt.Errorf(tarsMetaV1beta2.ResourceInvalidError, tserver, "no default node image has been set")
				}

				jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
					OP:    tarsMetaTools.JsonPatchAdd,
					Path:  "/spec/release/nodeImage",
					Value: image,
				})

				jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
					OP:    tarsMetaTools.JsonPatchAdd,
					Path:  "/spec/release/nodeSecret",
					Value: secret,
				})
			}
		}

		if tserver.Spec.Normal != nil {
			if tserver.Spec.Release.TServerReleaseNode != nil {
				if tserver.Spec.Release.TServerReleaseNode.Image != "" {
					jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
						OP:   tarsMetaTools.JsonPatchRemove,
						Path: "/spec/release/nodeImage",
					})
				}

				if tserver.Spec.Release.TServerReleaseNode.Secret != "" {
					jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
						OP:   tarsMetaTools.JsonPatchRemove,
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
