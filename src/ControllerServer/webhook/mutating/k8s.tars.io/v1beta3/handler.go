package v1beta3

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"tarscontroller/controller"
)

var functions map[string]func(*k8sAdmissionV1.AdmissionReview) ([]byte, error)

func init() {
	functions = map[string]func(*k8sAdmissionV1.AdmissionReview) ([]byte, error){

		"CREATE/TServer": mutatingCreateTServer,
		"UPDATE/TServer": mutatingUpdateTServer,

		"CREATE/TConfig": mutatingCreateTConfig,
		"UPDATE/TConfig": mutatingUpdateTConfig,

		"CREATE/TTree": mutatingCreateTTree,
		"UPDATE/TTree": mutatingUpdateTTree,

		"CREATE/TAccount": mutatingCreateTAccount,
		"UPDATE/TAccount": mutatingUpdateTAccount,

		"CREATE/TImage": mutatingCreateTImage,
		"UPDATE/TImage": mutatingUpdateTImage,

		"CREATE/TTemplate": mutatingCreateTTemplate,
		"UPDATE/TTemplate": mutatingUpdateTTemplate,
	}

}

func Handle(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	key := fmt.Sprintf("%s/%s", string(view.Request.Operation), view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(view)
	}
	return nil, fmt.Errorf("unsupported mutating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}
