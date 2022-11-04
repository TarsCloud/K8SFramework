package validating

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsMeta "k8s.tars.io/meta"
	"net/http"
	"tarswebhook/webhook/informer"
	validatingAppsV1 "tarswebhook/webhook/validating/apps/v1"
	validatingCoreV1 "tarswebhook/webhook/validating/core/v1"
	validatingAppsV1Beta2 "tarswebhook/webhook/validating/k8s.tars.io/v1beta2"
	validatingAppsV1Beta3 "tarswebhook/webhook/validating/k8s.tars.io/v1beta3"
)

type Validating struct {
	listers *informer.Listers
}

func New(listers *informer.Listers) *Validating {
	return &Validating{
		listers: listers,
	}
}

var handlers = map[string]func(*informer.Listers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	handlers = map[string]func(*informer.Listers, *k8sAdmissionV1.AdmissionReview) error{
		"core/v1":                     validatingCoreV1.Handler,
		"/v1":                         validatingCoreV1.Handler,
		"apps/v1":                     validatingAppsV1.Handler,
		tarsMeta.TarsGroupVersionV1B2: validatingAppsV1Beta2.Handler,
		tarsMeta.TarsGroupVersionV1B3: validatingAppsV1Beta3.Handler,
	}
}

func (v *Validating) Handle(w http.ResponseWriter, r *http.Request) {
	requestView := &k8sAdmissionV1.AdmissionReview{}

	err := json.NewDecoder(r.Body).Decode(requestView)
	if err != nil {
		return
	}

	gv := fmt.Sprintf("%s/%s", requestView.Request.Kind.Group, requestView.Request.Kind.Version)
	if fun, ok := handlers[gv]; !ok {
		err = fmt.Errorf("unsupported validating %s.%s", gv, requestView.Request.Kind.Kind)
	} else {
		err = fun(v.listers, requestView)
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