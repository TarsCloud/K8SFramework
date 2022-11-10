package mutating

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

type Mutator func(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) ([]byte, error)

var handlers = map[string]Mutator{}

func generateKey(operator k8sAdmissionV1.Operation, gvr string) string {
	return fmt.Sprintf("%s %s", operator, gvr)
}

func Registry(operator k8sAdmissionV1.Operation, gvr *schema.GroupVersionResource, handler Mutator) {
	key := generateKey(operator, gvr.String())
	klog.Infof("registry mutating key: [%s]\n", key)
	handlers[key] = handler
}

type Mutating struct {
	listers *lister.Listers
}

func New(listers *lister.Listers) *Mutating {
	return &Mutating{
		listers: listers,
	}
}

func (v *Mutating) Handle(w http.ResponseWriter, r *http.Request) {

	requestView := &k8sAdmissionV1.AdmissionReview{}
	err := json.NewDecoder(r.Body).Decode(&requestView)
	if err != nil {
		return
	}

	var patchContent []byte
	key := generateKey(requestView.Request.Operation, requestView.Request.RequestResource.String())
	klog.Infof("receiver mutating request, key is [%s]", key)
	if mutator, ok := handlers[key]; !ok || mutator == nil {
		err = fmt.Errorf("unsupported mutating [%s]", key)
	} else {
		patchContent, err = mutator(v.listers, requestView)
	}

	responseAdmissionView := k8sAdmissionV1.AdmissionReview{
		TypeMeta: k8sMetaV1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: k8sAdmissionV1.SchemeGroupVersion.String(),
		},
		Response: &k8sAdmissionV1.AdmissionResponse{
			UID: requestView.Request.UID,
		},
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
