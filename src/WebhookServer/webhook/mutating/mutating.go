package mutating

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsMeta "k8s.tars.io/meta"
	"net/http"
	"tarswebhook/webhook/informer"
	appsMutatingV1beta2 "tarswebhook/webhook/mutating/k8s.tars.io/v1beta2"
	appsMutatingV1beta3 "tarswebhook/webhook/mutating/k8s.tars.io/v1beta3"
)

type Mutating struct {
	listers *informer.Listers
}

func New(listers *informer.Listers) *Mutating {
	return &Mutating{
		listers: listers,
	}
}

var handlers = map[string]func(*informer.Listers, *k8sAdmissionV1.AdmissionReview) ([]byte, error){}

func init() {
	handlers = map[string]func(*informer.Listers, *k8sAdmissionV1.AdmissionReview) ([]byte, error){
		tarsMeta.TarsGroupVersionV1B2: appsMutatingV1beta2.Handle,
		tarsMeta.TarsGroupVersionV1B3: appsMutatingV1beta3.Handle,
	}
}

func (v *Mutating) Handle(w http.ResponseWriter, r *http.Request) {

	requestView := &k8sAdmissionV1.AdmissionReview{}
	err := json.NewDecoder(r.Body).Decode(&requestView)
	if err != nil {
		return
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

	var patchContent []byte
	gv := fmt.Sprintf("%s/%s", requestView.Request.Kind.Group, requestView.Request.Kind.Version)
	if fun, ok := handlers[gv]; !ok {
		err = fmt.Errorf("unsupported mutating %s.%s", gv, requestView.Request.Kind.Kind)
	} else {
		patchContent, err = fun(v.listers, requestView)
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
