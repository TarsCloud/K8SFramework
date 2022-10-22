package v1beta2

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"tarscontroller/util"
	"tarscontroller/webhook/informer"
)

var functions = map[string]func(*util.Clients, *informer.Listers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*util.Clients, *informer.Listers, *k8sAdmissionV1.AdmissionReview) error{

		"CREATE/TServer": validCreateTServer,
		"UPDATE/TServer": validUpdateTServer,
		"DELETE/TServer": validDeleteTServer,

		"CREATE/TConfig": validCreateTConfig,
		"UPDATE/TConfig": validUpdateTConfig,
		"DELETE/TConfig": validDeleteTConfig,

		"CREATE/TTemplate": validCreateTTemplate,
		"UPDATE/TTemplate": validUpdateTTemplate,
		"DELETE/TTemplate": validDeleteTTemplate,

		"CREATE/TTree": validCreateTTree,
		"UPDATE/TTree": validUpdateTTree,
		"DELETE/TTree": validDeleteTTree,

		"CREATE/TAccount": validCreateTAccount,
		"UPDATE/TAccount": validUpdateTAccount,
		"DELETE/TAccount": validDeleteTAccount,
	}
}

func Handler(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(clients, listers, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}
