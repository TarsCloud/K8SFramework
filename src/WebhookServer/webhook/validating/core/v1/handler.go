package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"tarswebhook/webhook/informer"
)

var functions = map[string]func(*informer.Listers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*informer.Listers, *k8sAdmissionV1.AdmissionReview) error{
		"CREATE/Service": validCreateService,
		"UPDATE/Service": validUpdateService,
		"DELETE/Service": validDeleteService,
	}
}

func Handler(listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(listers, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}
