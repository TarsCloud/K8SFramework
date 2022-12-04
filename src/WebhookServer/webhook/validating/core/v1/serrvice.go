package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"tarswebhook/webhook/lister"
	"tarswebhook/webhook/validating"
)

func validService(newService *k8sCoreV1.Service, oldService *k8sCoreV1.Service, listers *lister.Listers) error {
	if !listers.TSSynced() {
		return fmt.Errorf("tserver infomer has not finished syncing")
	}

	namespace := newService.Namespace
	tserver, err := listers.TSLister.TServers(namespace).Get(newService.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newService.Name, err.Error())
	}

	equal, _ := tarsRuntime.TarsTranslator.DryRunSyncService(tserver, newService)
	if !equal {
		return fmt.Errorf("resource should be modified through tserver")
	}

	return nil
}

func validCreateService(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	return fmt.Errorf("only use authorized account can create service")
}

func validUpdateService(informer *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	newService := &k8sCoreV1.Service{}
	_ = json.Unmarshal(view.Request.Object.Raw, newService)

	return validService(newService, nil, informer)
}

func validDeleteService(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func init() {
	gvr := k8sCoreV1.SchemeGroupVersion.WithResource("services")
	validating.Registry(k8sAdmissionV1.Create, &gvr, validCreateService)
	validating.Registry(k8sAdmissionV1.Update, &gvr, validUpdateService)
	validating.Registry(k8sAdmissionV1.Delete, &gvr, validDeleteService)
}
