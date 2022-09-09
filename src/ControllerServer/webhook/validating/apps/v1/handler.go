package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"tarscontroller/controller"
)

var functions = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) error{
		"CREATE/StatefulSet": validCreateStatefulSet,
		"UPDATE/StatefulSet": validUpdateStatefulSet,
		"DELETE/StatefulSet": validDeleteStatefulSet,

		"CREATE/DaemonSet": validCreateDaemonSet,
		"UPDATE/DaemonSet": validUpdateDaemonSet,
		"DELETE/DaemonSet": validDeleteDaemonSet,
	}
}

func Handler(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(clients, informers, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}
