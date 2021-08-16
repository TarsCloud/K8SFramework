package v1

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"tarscontroller/meta"
)

var functions = map[string]func(*meta.Clients, *meta.Informers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*meta.Clients, *meta.Informers, *k8sAdmissionV1.AdmissionReview) error{
		"CREATE/Service": validCreateService,
		"UPDATE/Service": validUpdateService,
		"DELETE/Service": validDeleteService,
	}
}

type Handler struct {
	clients  *meta.Clients
	informer *meta.Informers
}

func New(clients *meta.Clients, informers *meta.Informers) *Handler {
	return &Handler{clients: clients, informer: informers}
}

func (v *Handler) Handle(view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(v.clients, v.informer, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}

func validService(newService *k8sCoreV1.Service, oldService *k8sCoreV1.Service, clients *meta.Clients, informer *meta.Informers) error {
	namespace := newService.Namespace
	tserver, err := informer.TServerInformer.Lister().TServers(namespace).Get(newService.Name)
	if err != nil {
		return fmt.Errorf(meta.ResourceGetError, "tserver", namespace, newService.Name, err.Error())
	}

	if !meta.EqualTServerAndService(tserver, newService) {
		return fmt.Errorf("resource should be modified through tserver")
	}

	return nil
}

func validCreateService(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	if view.Request.UserInfo.Username == meta.GetControllerUsername() {
		return nil
	}
	return fmt.Errorf("only use authorized account can create service")
}

func validUpdateService(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	if view.Request.UserInfo.Username == meta.GetControllerUsername() {
		return nil
	}
	newService := &k8sCoreV1.Service{}
	_ = json.Unmarshal(view.Request.Object.Raw, newService)
	return validService(newService, nil, clients, informer)
}

func validDeleteService(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
