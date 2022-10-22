package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"tarscontroller/util"
	"tarscontroller/webhook/informer"
)

var functions = map[string]func(*util.Clients, *informer.Listers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*util.Clients, *informer.Listers, *k8sAdmissionV1.AdmissionReview) error{
		"CREATE/StatefulSet": validCreateStatefulSet,
		"UPDATE/StatefulSet": validUpdateStatefulSet,
		"DELETE/StatefulSet": validDeleteStatefulSet,

		"CREATE/DaemonSet": validCreateDaemonSet,
		"UPDATE/DaemonSet": validUpdateDaemonSet,
		"DELETE/DaemonSet": validDeleteDaemonSet,
	}
}

func Handler(clients *util.Clients, informers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(clients, informers, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}
