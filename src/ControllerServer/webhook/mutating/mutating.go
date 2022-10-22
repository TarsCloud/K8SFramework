package mutating

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsMeta "k8s.tars.io/meta"
	"net/http"
	"tarscontroller/util"
	"tarscontroller/webhook/informer"
	crdMutatingV1beta2 "tarscontroller/webhook/mutating/k8s.tars.io/v1beta2"
	crdMutatingV1beta3 "tarscontroller/webhook/mutating/k8s.tars.io/v1beta3"
)

type Mutating struct {
	clients *util.Clients
	listers *informer.Listers
}

func New(clients *util.Clients, listers *informer.Listers) *Mutating {
	return &Mutating{
		clients: clients,
		listers: listers,
	}
}

var handlers = map[string]func(*util.Clients, *informer.Listers, *k8sAdmissionV1.AdmissionReview) ([]byte, error){}

func init() {
	handlers = map[string]func(*util.Clients, *informer.Listers, *k8sAdmissionV1.AdmissionReview) ([]byte, error){
		tarsMeta.TarsGroupVersionV1B2: crdMutatingV1beta2.Handle,
		tarsMeta.TarsGroupVersionV1B3: crdMutatingV1beta3.Handle,
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
		patchContent, err = fun(v.clients, v.listers, requestView)
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
