package v1

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	crdMeta "k8s.tars.io/api/meta"
	"tarscontroller/controller"
	"tarscontroller/reconcile/v1beta2"
)

var functions = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) error{
		"CREATE/Service": validCreateService,
		"UPDATE/Service": validUpdateService,
		"DELETE/Service": validDeleteService,
	}
}

type Handler struct {
	clients  *controller.Clients
	informer *controller.Informers
}

func New(clients *controller.Clients, informers *controller.Informers) *Handler {
	return &Handler{clients: clients, informer: informers}
}

func (v *Handler) Handle(view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(v.clients, v.informer, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}

func validService(newService *k8sCoreV1.Service, oldService *k8sCoreV1.Service, clients *controller.Clients, informer *controller.Informers) error {
	namespace := newService.Namespace
	tserver, err := informer.TServerInformer.Lister().TServers(namespace).Get(newService.Name)
	if err != nil {
		return fmt.Errorf(crdMeta.ResourceGetError, "tserver", namespace, newService.Name, err.Error())
	}

	if !v1beta2.EqualTServerAndService(tserver, newService) {
		return fmt.Errorf("resource should be modified through tserver")
	}

	return nil
}

func validCreateService(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	return fmt.Errorf("only use authorized account can create service")
}

func validUpdateService(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newService := &k8sCoreV1.Service{}
	_ = json.Unmarshal(view.Request.Object.Raw, newService)
	return validService(newService, nil, clients, informer)
}

func validDeleteService(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
