package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sAppsV1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/controller"
	tarsCrdV1beta3 "tarscontroller/reconcile/v1beta3"
)

func validStatefulSet(newStatefulset *k8sAppsV1.StatefulSet, oldStatefulset *k8sAppsV1.StatefulSet, clients *controller.Clients, informer *controller.Informers) error {
	namespace := newStatefulset.Namespace
	tserver, err := informer.TServerInformer.Lister().TServers(namespace).Get(newStatefulset.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newStatefulset.Name, err.Error())
	}
	if !tarsCrdV1beta3.EqualTServerAndStatefulSet(tserver, newStatefulset) {
		return fmt.Errorf("resource should be modified through tserver")
	}
	return nil
}

func validCreateStatefulSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	return fmt.Errorf("only use authorized account can create statefulset")
}

func validUpdateStatefulSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newStatefulset := &k8sAppsV1.StatefulSet{}
	_ = json.Unmarshal(view.Request.Object.Raw, newStatefulset)
	return validStatefulSet(newStatefulset, nil, clients, informer)
}

func validDeleteStatefulSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
