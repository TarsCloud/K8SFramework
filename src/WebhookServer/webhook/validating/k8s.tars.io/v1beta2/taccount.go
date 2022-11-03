package v1beta2

import (
	"crypto/md5"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsAppsV1beta2 "k8s.tars.io/apps/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"tarswebhook/webhook/informer"
)

func validTAccount(newTAccount *tarsAppsV1beta2.TAccount, oldTAccount *tarsAppsV1beta2.TAccount, listers *informer.Listers) error {
	expectedResourceName := fmt.Sprintf("%x", md5.Sum([]byte(newTAccount.Spec.Username)))
	if newTAccount.Name != expectedResourceName {
		return fmt.Errorf(tarsMeta.ResourceInvalidError, "taccount", "unexpected resource name")
	}
	return nil
}

func validCreateTAccount(listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTAccount := &tarsAppsV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)
	return validTAccount(newTAccount, nil, listers)
}

func validUpdateTAccount(listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := tarsRuntime.Username
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	newTAccount := &tarsAppsV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)

	oldTAccount := &tarsAppsV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTAccount)

	return validTAccount(newTAccount, oldTAccount, listers)
}

func validDeleteTAccount(listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
