package v1beta2

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1beta2 "k8s.tars.io/crd/v1beta2"
	tarsMeta "k8s.tars.io/meta"
)

func mutatingCreateTTree(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTTree := &tarsCrdV1beta2.TTree{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTTree)

	businessMap := make(map[string]interface{}, len(newTTree.Businesses))
	for _, business := range newTTree.Businesses {
		businessMap[business.Name] = nil
	}

	var jsonPatch tarsMeta.JsonPatch

	for i, app := range newTTree.Apps {
		if app.BusinessRef != "" {
			if _, ok := businessMap[app.BusinessRef]; !ok {
				jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
					OP:    tarsMeta.JsonPatchReplace,
					Path:  fmt.Sprintf("/apps/%d/businessRef", i),
					Value: "",
				})
			}
		}
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTTree(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTTree(requestAdmissionView)
}
