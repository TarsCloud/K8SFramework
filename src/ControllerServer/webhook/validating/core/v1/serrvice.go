package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/controller"
	"tarscontroller/reconcile/v1beta3"
)

func validService(newService *k8sCoreV1.Service, oldService *k8sCoreV1.Service, clients *controller.Clients, informer *controller.Informers) error {
	namespace := newService.Namespace
	tserver, err := informer.TServerInformer.Lister().TServers(namespace).Get(newService.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newService.Name, err.Error())
	}

	if !v1beta3.EqualTServerAndService(tserver, newService) {
		return fmt.Errorf("resource should be modified through tserver")
	}

	return nil
}

func validCreateService(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	return fmt.Errorf("only use authorized account can create service")
}

func validUpdateService(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newService := &k8sCoreV1.Service{}
	_ = json.Unmarshal(view.Request.Object.Raw, newService)
	return validService(newService, nil, clients, informer)
}

func validDeleteService(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
