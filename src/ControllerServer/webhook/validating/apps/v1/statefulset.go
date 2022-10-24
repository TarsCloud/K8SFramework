package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sAppsV1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsMeta "k8s.tars.io/meta"
	tarsControllerV1beta3 "tarscontroller/controller/v1beta3"
	"tarscontroller/util"
	"tarscontroller/webhook/informer"
)

func validStatefulSet(newStatefulset *k8sAppsV1.StatefulSet, oldStatefulset *k8sAppsV1.StatefulSet, clients *util.Clients, listers *informer.Listers) error {
	if !listers.TSSynced() {
		return fmt.Errorf("tserver infomer has not finished syncing")
	}

	namespace := newStatefulset.Namespace
	tserver, err := listers.TSLister.TServers(namespace).Get(newStatefulset.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newStatefulset.Name, err.Error())
	}

	if !tarsControllerV1beta3.EqualTServerAndStatefulSet(tserver, newStatefulset) {
		return fmt.Errorf("resource should be modified through tserver")
	}
	return nil
}

func validCreateStatefulSet(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := util.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	return fmt.Errorf("only use authorized account can create statefulset")
}

func validUpdateStatefulSet(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := util.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	newStatefulset := &k8sAppsV1.StatefulSet{}
	_ = json.Unmarshal(view.Request.Object.Raw, newStatefulset)
	return validStatefulSet(newStatefulset, nil, clients, listers)
}

func validDeleteStatefulSet(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
