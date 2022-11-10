package v1beta3

import (
	"crypto/md5"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"tarswebhook/webhook/lister"
	"tarswebhook/webhook/validating"
)

func validTAccount(newTAccount *tarsV1beta3.TAccount, oldTAccount *tarsV1beta3.TAccount, listers *lister.Listers) error {
	expectedResourceName := fmt.Sprintf("%x", md5.Sum([]byte(newTAccount.Spec.Username)))
	if newTAccount.Name != expectedResourceName {
		return fmt.Errorf(tarsMeta.ResourceInvalidError, "taccount", "unexpected resource name")
	}
	return nil
}

func validCreateTAccount(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTAccount := &tarsV1beta3.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)
	return validTAccount(newTAccount, nil, listers)
}

func validUpdateTAccount(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	newTAccount := &tarsV1beta3.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)

	oldTAccount := &tarsV1beta3.TAccount{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTAccount)

	return validTAccount(newTAccount, oldTAccount, listers)
}

func validDeleteTAccount(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func init() {
	gvr := tarsV1beta3.SchemeGroupVersion.WithResource("taccounts")
	validating.Registry(k8sAdmissionV1.Create, &gvr, validCreateTAccount)
	validating.Registry(k8sAdmissionV1.Update, &gvr, validUpdateTAccount)
	validating.Registry(k8sAdmissionV1.Delete, &gvr, validDeleteTAccount)
}
