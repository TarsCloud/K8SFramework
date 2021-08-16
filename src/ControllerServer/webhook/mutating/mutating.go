package mutating

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"tarscontroller/meta"
	crdV1alpha1 "tarscontroller/webhook/mutating/k8s.tars.io/v1alpha1"
	crdV1alpha2 "tarscontroller/webhook/mutating/k8s.tars.io/v1alpha2"
)

type Mutating struct {
	crdV1alpha1Handler *crdV1alpha1.Handler
	crdV1alpha2Handler *crdV1alpha2.Handler
}

func New(clients *meta.Clients, informers *meta.Informers) *Mutating {
	v := &Mutating{
		crdV1alpha1Handler: crdV1alpha1.New(clients, informers),
		crdV1alpha2Handler: crdV1alpha2.New(clients, informers),
	}
	return v
}

func (v Mutating) Handle(w http.ResponseWriter, r *http.Request) {

	requestView := &k8sAdmissionV1.AdmissionReview{}
	err := json.NewDecoder(r.Body).Decode(&requestView)
	if err != nil {
		return
	}

	responseAdmissionView := k8sAdmissionV1.AdmissionReview{
		TypeMeta: k8sMetaV1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &k8sAdmissionV1.AdmissionResponse{
			UID: requestView.Request.UID,
		},
	}

	var patchContent []byte
	gv := fmt.Sprintf("%s/%s", requestView.Request.Kind.Group, requestView.Request.Kind.Version)
	switch gv {
	case "k8s.tars.io/v1alpha1":
		patchContent, err = v.crdV1alpha1Handler.Handle(requestView)
	case "k8s.tars.io/v1alpha2":
		patchContent, err = v.crdV1alpha2Handler.Handle(requestView)
	default:
		err = fmt.Errorf("unsupported mutating %s.%s", gv, requestView.Request.Kind.Kind)
	}

	if err != nil {
		responseAdmissionView.Response.Allowed = false
		responseAdmissionView.Response.Result = &k8sMetaV1.Status{
			Status:  "Failure",
			Message: err.Error(),
		}
		responseBytes, _ := json.Marshal(responseAdmissionView)
		_, _ = w.Write(responseBytes)
		return
	}

	responseAdmissionView.Response.Allowed = true
	if patchContent != nil {
		responseAdmissionView.Response.Patch = patchContent
		patchType := k8sAdmissionV1.PatchTypeJSONPatch
		responseAdmissionView.Response.PatchType = &patchType
	}
	responseBytes, _ := json.Marshal(responseAdmissionView)
	_, _ = w.Write(responseBytes)
}
