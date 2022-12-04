package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sAppsV1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"tarswebhook/webhook/lister"
	"tarswebhook/webhook/validating"
)

func validDaemonset(newDaemonset, oldDaemonset *k8sAppsV1.DaemonSet, listers *lister.Listers) error {
	if !listers.TSSynced() {
		return fmt.Errorf("tserver infomer has not finished syncing")
	}

	namespace := newDaemonset.Namespace
	tserver, err := listers.TSLister.TServers(namespace).Get(newDaemonset.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newDaemonset.Name, err.Error())
	}

	equal, _ := tarsRuntime.TarsTranslator.DryRunSyncDaemonset(tserver, newDaemonset)
	if !equal {
		return fmt.Errorf("resource should be modified through tserver")
	}

	return nil
}

func validCreateDaemonSet(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}
	return fmt.Errorf("only use authorized account can create daemonset")
}

func validUpdateDaemonSet(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	newDaemonset := &k8sAppsV1.DaemonSet{}
	_ = json.Unmarshal(view.Request.Object.Raw, newDaemonset)

	return validDaemonset(newDaemonset, nil, listers)
}

func validDeleteDaemonSet(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func init() {
	gvr := k8sAppsV1.SchemeGroupVersion.WithResource("daemonsets")
	validating.Registry(k8sAdmissionV1.Create, &gvr, validCreateDaemonSet)
	validating.Registry(k8sAdmissionV1.Update, &gvr, validUpdateDaemonSet)
	validating.Registry(k8sAdmissionV1.Delete, &gvr, validDeleteDaemonSet)
}
