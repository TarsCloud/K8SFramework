package validating

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"net/http"
	"tarswebhook/webhook/lister"
)

type Validator func(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error

var handlers = map[string]Validator{}

func generateKey(operator k8sAdmissionV1.Operation, gvr string) string {
	return fmt.Sprintf("%s %s", operator, gvr)
}

func Registry(operator k8sAdmissionV1.Operation, gvr *schema.GroupVersionResource, handler Validator) {
	key := generateKey(operator, gvr.String())
	klog.Infof("registry validating key: [%s]\n", key)
	handlers[key] = handler
}

type Validating struct {
	listers *lister.Listers
}

func New(listers *lister.Listers) *Validating {
	return &Validating{
		listers: listers,
	}
}

func (v *Validating) Handle(w http.ResponseWriter, r *http.Request) {
	requestView := &k8sAdmissionV1.AdmissionReview{}

	err := json.NewDecoder(r.Body).Decode(requestView)
	if err != nil {
		return
	}

	key := generateKey(requestView.Request.Operation, requestView.Request.RequestResource.String())
	klog.Infof("receiver validating request, key is [%s]", key)
	if validator, ok := handlers[key]; !ok || validator == nil {
		err = fmt.Errorf("unsupported validating [%s]", key)
	} else {
		err = validator(v.listers, requestView)
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
