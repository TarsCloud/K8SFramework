package validating

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"tarscontroller/controller"
	appsV1 "tarscontroller/webhook/validating/apps/v1"
	coreV1 "tarscontroller/webhook/validating/core/v1"
	crdV1beta2 "tarscontroller/webhook/validating/k8s.tars.io/v1beta2"
)

type Validating struct {
	crdV1beta2Handler *crdV1beta2.Handler
	coreV1Handler     *coreV1.Handler
	appsV1Handler     *appsV1.Handler
}

func New(clients *controller.Clients, informers *controller.Informers) *Validating {
	v := &Validating{
		crdV1beta2Handler: crdV1beta2.New(clients, informers),
		coreV1Handler:     coreV1.New(clients, informers),
		appsV1Handler:     appsV1.New(clients, informers),
	}
	return v
}

func (v *Validating) Handle(w http.ResponseWriter, r *http.Request) {
	requestView := &k8sAdmissionV1.AdmissionReview{}

	err := json.NewDecoder(r.Body).Decode(requestView)
	if err != nil {
		return
	}

	gv := fmt.Sprintf("%s/%s", requestView.Request.Kind.Group, requestView.Request.Kind.Version)
	switch gv {
	case "k8s.tars.io/v1beta2":
		err = v.crdV1beta2Handler.Handle(requestView)
	case "apps/v1":
		err = v.appsV1Handler.Handle(requestView)
	case "/v1", "core/v1":
		err = v.coreV1Handler.Handle(requestView)
	default:
		err = fmt.Errorf("unsupported validating %s.%s", gv, requestView.Request.Kind.Kind)
	}

	var responseView = &k8sAdmissionV1.AdmissionReview{
		TypeMeta: k8sMetaV1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: &k8sAdmissionV1.AdmissionResponse{
			UID: requestView.Request.UID,
		},
	}
	if err != nil {
		responseView.Response.Allowed = false
		responseView.Response.Result = &k8sMetaV1.Status{
			Status:  "Failure",
			Message: err.Error(),
		}
	} else {
		responseView.Response.Allowed = true
	}
	responseBytes, _ := json.Marshal(responseView)
	_, _ = w.Write(responseBytes)
}
