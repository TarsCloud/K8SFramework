package v1beta2

import (
	"crypto/md5"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1beta2 "k8s.tars.io/crd/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/controller"
)

func validTAccount(newTAccount *tarsCrdV1beta2.TAccount, oldTAccount *tarsCrdV1beta2.TAccount, client *controller.Clients, informers *controller.Informers) error {
	expectedResourceName := fmt.Sprintf("%x", md5.Sum([]byte(newTAccount.Spec.Username)))
	if newTAccount.Name != expectedResourceName {
		return fmt.Errorf(tarsMeta.ResourceInvalidError, "taccount", "unexpected resource name")
	}
	return nil
}

func validCreateTAccount(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTAccount := &tarsCrdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)
	return validTAccount(newTAccount, nil, clients, informers)
}

func validUpdateTAccount(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	newTAccount := &tarsCrdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)

	oldTAccount := &tarsCrdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTAccount)

	return validTAccount(newTAccount, oldTAccount, clients, informers)
}

func validDeleteTAccount(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}
