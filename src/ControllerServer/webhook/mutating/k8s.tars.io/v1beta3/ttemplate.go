package v1beta3

import (
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMetaTools "k8s.tars.io/meta/tools"
	tarsMetaV1beta3 "k8s.tars.io/meta/v1beta3"
)

func mutatingCreateTTemplate(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	ttemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, ttemplate)

	var jsonPatch tarsMetaTools.JsonPatch

	for i := 0; i < 1; i++ {

		fatherless := ttemplate.Name == ttemplate.Spec.Parent

		if fatherless {
			if ttemplate.Labels != nil {
				if _, ok := ttemplate.Labels[tarsMetaV1beta3.ParentLabel]; ok {
					jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
						OP:   tarsMetaTools.JsonPatchRemove,
						Path: "/metadata/labels/tars.io~1Parent",
					})
				}
			}
			break
		}
		if ttemplate.Labels == nil {
			labels := map[string]string{}
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/metadata/labels",
				Value: labels,
			})
		}

		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Parent",
			Value: ttemplate.Spec.Parent,
		})
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}

	return nil, nil
}

func mutatingUpdateTTemplate(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTTemplate(requestAdmissionView)
}
