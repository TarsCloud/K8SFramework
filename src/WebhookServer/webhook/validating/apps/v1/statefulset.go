package v1

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sAppsV1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsMeta "k8s.tars.io/meta"
	translatorV1beta3 "k8s.tars.io/translator/v1beta3"
	"tarswebhook/webhook/lister"
	"tarswebhook/webhook/validating"
)

func validStatefulSet(newStatefulset *k8sAppsV1.StatefulSet, oldStatefulset *k8sAppsV1.StatefulSet, listers *lister.Listers) error {
	if !listers.TSSynced() {
		return fmt.Errorf("tserver infomer has not finished syncing")
	}

	namespace := newStatefulset.Namespace
	tserver, err := listers.TSLister.TServers(namespace).Get(newStatefulset.Name)
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, newStatefulset.Name, err.Error())
	}

	translator := translatorV1beta3.Translator{}
	equal, _ := translator.DryRunSyncStatefulset(tserver, newStatefulset)
	if !equal {
		return fmt.Errorf("resource should be modified through tserver")
	}
	return nil
}

func validCreateStatefulSet(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	return fmt.Errorf("only use authorized account can create statefulset")
}

func validUpdateStatefulSet(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	newStatefulset := &k8sAppsV1.StatefulSet{}
	_ = json.Unmarshal(view.Request.Object.Raw, newStatefulset)
	return validStatefulSet(newStatefulset, nil, listers)
}

func validDeleteStatefulSet(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func init() {
	gvr := k8sAppsV1.SchemeGroupVersion.WithResource("statefulsets")
	validating.Registry(k8sAdmissionV1.Create, &gvr, validCreateStatefulSet)
	validating.Registry(k8sAdmissionV1.Update, &gvr, validUpdateStatefulSet)
	validating.Registry(k8sAdmissionV1.Delete, &gvr, validDeleteStatefulSet)
}
