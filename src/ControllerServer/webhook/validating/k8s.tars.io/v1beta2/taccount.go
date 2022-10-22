package v1beta2

import (
	"crypto/md5"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1beta2 "k8s.tars.io/crd/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/util"
	"tarscontroller/webhook/informer"
)

func validTAccount(newTAccount *tarsCrdV1beta2.TAccount, oldTAccount *tarsCrdV1beta2.TAccount, client *util.Clients, listers *informer.Listers) error {
	expectedResourceName := fmt.Sprintf("%x", md5.Sum([]byte(newTAccount.Spec.Username)))
	if newTAccount.Name != expectedResourceName {
		return fmt.Errorf(tarsMeta.ResourceInvalidError, "taccount", "unexpected resource name")
	}
	return nil
}

func validCreateTAccount(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTAccount := &tarsCrdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)
	return validTAccount(newTAccount, nil, clients, listers)
}

func validUpdateTAccount(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := util.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	newTAccount := &tarsCrdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)

	oldTAccount := &tarsCrdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTAccount)

	return validTAccount(newTAccount, oldTAccount, clients, listers)
}

func validDeleteTAccount(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
