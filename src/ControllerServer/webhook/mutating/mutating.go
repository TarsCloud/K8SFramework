package mutating

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsMetaV1beta2 "k8s.tars.io/meta/v1beta2"
	tarsMetaV1beta3 "k8s.tars.io/meta/v1beta3"
	"net/http"
	"tarscontroller/controller"
	crdMutatingV1beta2 "tarscontroller/webhook/mutating/k8s.tars.io/v1beta2"
	crdMutatingV1beta3 "tarscontroller/webhook/mutating/k8s.tars.io/v1beta3"
)

type Mutating struct {
	clients   *controller.Clients
	informers *controller.Informers
}

func New(clients *controller.Clients, informers *controller.Informers) *Mutating {
	return &Mutating{
		clients:   clients,
		informers: informers,
	}
}

var handlers = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) ([]byte, error){}

func init() {
	handlers = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) ([]byte, error){
		tarsMetaV1beta2.GroupVersion: crdMutatingV1beta2.Handle,
		tarsMetaV1beta3.GroupVersion: crdMutatingV1beta3.Handle,
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
			APIVersion: "admission.k8s.io/v1",
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
		patchContent, err = fun(v.clients, v.informers, requestView)
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
