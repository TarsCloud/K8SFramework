package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsMeta "k8s.tars.io/meta"
	tarsControllerV1beta3 "tarscontroller/controller/v1beta3"
	"tarscontroller/util"
	"tarscontroller/webhook/informer"
)

func validService(newService *k8sCoreV1.Service, oldService *k8sCoreV1.Service, clients *util.Clients, listers *informer.Listers) error {
	if !listers.TSSynced() {
		return fmt.Errorf("tserver infomer has not finished syncing")
	}

	namespace := newService.Namespace
	tserver, err := listers.TSLister.TServers(namespace).Get(newService.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newService.Name, err.Error())
	}

	if !tarsControllerV1beta3.EqualTServerAndService(tserver, newService) {
		return fmt.Errorf("resource should be modified through tserver")
	}

	return nil
}

func validCreateService(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := util.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	return fmt.Errorf("only use authorized account can create service")
}

func validUpdateService(clients *util.Clients, informer *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := util.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	newService := &k8sCoreV1.Service{}
	_ = json.Unmarshal(view.Request.Object.Raw, newService)

	return validService(newService, nil, clients, informer)
}

func validDeleteService(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
