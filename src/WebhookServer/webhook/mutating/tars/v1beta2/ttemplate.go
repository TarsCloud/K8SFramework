package v1beta2

import (
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1beta2 "k8s.tars.io/apis/tars/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	tarsTool "k8s.tars.io/tool"
	"tarswebhook/webhook/lister"
	"tarswebhook/webhook/mutating"
)

func mutatingCreateTTemplate(listers *lister.Listers, requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	ttemplate := &tarsV1beta2.TTemplate{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, ttemplate)

	var jsonPatch tarsTool.JsonPatch

	for i := 0; i < 1; i++ {

		fatherless := ttemplate.Name == ttemplate.Spec.Parent

		if fatherless {
			if ttemplate.Labels != nil {
				if _, ok := ttemplate.Labels[tarsMeta.TTemplateParentLabel]; ok {
					jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
						OP:   tarsTool.JsonPatchRemove,
						Path: "/metadata/labels/tars.io~1Parent",
					})
				}
			}
			break
		}

		if ttemplate.Labels == nil {
			labels := map[string]string{}
			jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
				OP:    tarsTool.JsonPatchAdd,
				Path:  "/metadata/labels",
				Value: labels,
			})
		}

		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Parent",
			Value: ttemplate.Spec.Parent,
		})
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}

	return nil, nil
}

func mutatingUpdateTTemplate(listers *lister.Listers, requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTTemplate(listers, requestAdmissionView)
}

func init() {
	gvr := tarsV1beta2.SchemeGroupVersion.WithResource("ttemplates")
	mutating.Registry(k8sAdmissionV1.Create, &gvr, mutatingCreateTTemplate)
	mutating.Registry(k8sAdmissionV1.Update, &gvr, mutatingUpdateTTemplate)
}
