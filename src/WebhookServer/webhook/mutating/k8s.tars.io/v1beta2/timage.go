package v1beta2

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1beta2 "k8s.tars.io/apis/tars/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	"strings"
)

func mutatingCreateTImage(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	timage := &tarsV1beta2.TImage{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, timage)

	var jsonPatch tarsMeta.JsonPatch

	if timage.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
		OP:    tarsMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ImageType",
		Value: timage.ImageType,
	})

	shouldAddSupportedLabel := make(map[string]string, len(timage.SupportedType))

	if timage.ImageType == "base" && timage.SupportedType != nil {
		for _, v := range timage.SupportedType {
			shouldAddSupportedLabel[fmt.Sprintf("tars.io/Supported.%s", v)] = v
		}
	}

	for k := range timage.Labels {
		if _, ok := shouldAddSupportedLabel[k]; ok {
			delete(shouldAddSupportedLabel, k)
		} else {
			if strings.HasPrefix(k, "tars.io/Supported.") {
				v := strings.ReplaceAll(k, "tars.io/", "tars.io~1")
				jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
					OP:   tarsMeta.JsonPatchRemove,
					Path: fmt.Sprintf("/metadata/labels/%s", v),
				})
			}
		}
	}

	for _, v := range shouldAddSupportedLabel {
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  fmt.Sprintf("/metadata/labels/tars.io~1Supported.%s", v),
			Value: v,
		})
	}

	// if there is a duplicate id, we will keep the previous one
	existing := map[string]interface{}{}
	removes := map[int]interface{}{}
	for i, v := range timage.Releases {
		if _, ok := existing[v.ID]; ok {
			newSeqAfterRemove := i - len(removes)
			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:   tarsMeta.JsonPatchRemove,
				Path: fmt.Sprintf("/releases/%d", newSeqAfterRemove),
			})
			removes[i] = nil
		}
		existing[v.ID] = nil
	}

	now := k8sMetaV1.Now().ToUnstructured()
	for i, v := range timage.Releases {
		if v.CreateTime.IsZero() {
			if _, ok := removes[i]; !ok {
				newSeqAfterRemove := i
				if i > len(removes) {
					newSeqAfterRemove = i - len(removes)
				}
				jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
					OP:    tarsMeta.JsonPatchAdd,
					Path:  fmt.Sprintf("/releases/%d/createTime", newSeqAfterRemove),
					Value: now,
				})
			}
		}
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTImage(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTImage(requestAdmissionView)
}
