package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sAppsV1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsMeta "k8s.tars.io/meta"
	tarsCrdV1beta3 "tarscontroller/controller/v1beta3"
	"tarscontroller/util"
	"tarscontroller/webhook/informer"
)

func validDaemonset(newDaemonset, oldDaemonset *k8sAppsV1.DaemonSet, clients *util.Clients, listers *informer.Listers) error {
	if !listers.TSSynced() {
		return fmt.Errorf("tserver infomer has not finished syncing")
	}

	namespace := newDaemonset.Namespace
	tserver, err := listers.TSLister.TServers(namespace).Get(newDaemonset.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newDaemonset.Name, err.Error())
	}

	if !tarsCrdV1beta3.EqualTServerAndDaemonSet(tserver, newDaemonset) {
		return fmt.Errorf("this resource should be modified through tserver")
	}

	return nil
}

func validCreateDaemonSet(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := util.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	return fmt.Errorf("only use authorized account can create daemonset")
}

func validUpdateDaemonSet(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := util.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	newDaemonset := &k8sAppsV1.DaemonSet{}
	_ = json.Unmarshal(view.Request.Object.Raw, newDaemonset)

	return validDaemonset(newDaemonset, nil, clients, listers)
}

func validDeleteDaemonSet(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
