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

func validDaemonset(newDaemonset, oldDaemonset *k8sAppsV1.DaemonSet, clients *controller.Clients, informer *controller.Informers) error {
	namespace := newDaemonset.Namespace
	tserver, err := informer.TServerInformer.Lister().TServers(namespace).Get(newDaemonset.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newDaemonset.Name, err.Error())
	}
	if !tarsCrdV1beta3.EqualTServerAndDaemonSet(tserver, newDaemonset) {
		return fmt.Errorf("this resource should be modified through tserver")
	}
	return nil
}

func validCreateDaemonSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	return fmt.Errorf("only use authorized account can create daemonset")
}

func validUpdateDaemonSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newDaemonset := &k8sAppsV1.DaemonSet{}
	_ = json.Unmarshal(view.Request.Object.Raw, newDaemonset)
	return validDaemonset(newDaemonset, nil, clients, informer)
}

func validDeleteDaemonSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
