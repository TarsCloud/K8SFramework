package v1beta3

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarscontroller/util"
	"tarscontroller/webhook/informer"
)

func validTTemplate(newTTemplate *tarsCrdV1beta3.TTemplate, oldTTemplate *tarsCrdV1beta3.TTemplate, clients *util.Clients, listers *informer.Listers) error {

	parentName := newTTemplate.Spec.Parent
	if parentName == "" {
		return fmt.Errorf(tarsMeta.ResourceInvalidError, "ttemplate", "value of filed \".spec.parent\" should not empty ")
	}

	if newTTemplate.Name == newTTemplate.Spec.Parent {
		return nil
	}

	if !listers.TTSynced() {
		return fmt.Errorf("ttemplate infomer has not finished syncing")
	}

	namespace := newTTemplate.Namespace
	_, err := listers.TTLister.ByNamespace(namespace).Get(parentName)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf(tarsMeta.ResourceGetError, "ttemplate", namespace, parentName, err.Error())
		}
		return fmt.Errorf(tarsMeta.ResourceNotExistError, "ttemplate", namespace, parentName)
	}

	return nil
}

func validCreateTTemplate(clients *util.Clients, informers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)
	return validTTemplate(newTTemplate, nil, clients, informers)
}

func validUpdateTTemplate(clients *util.Clients, informers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)

	oldTTemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTemplate)

	return validTTemplate(newTTemplate, oldTTemplate, clients, informers)
}

func validDeleteTTemplate(clients *util.Clients, listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	username := view.Request.UserInfo.Username
	controllerUserName := util.GetControllerUsername()

	if controllerUserName == username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	if strings.HasPrefix(username, tarsMeta.KubernetesSystemAccountPrefix) {
		return nil
	}

	ttemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttemplate)
	namespace := ttemplate.Namespace

	if !listers.TSSynced() {
		return fmt.Errorf("tserver infomer has not finished syncing")
	}

	requirement, _ := labels.NewRequirement(tarsMeta.TTemplateLabel, selection.DoubleEquals, []string{ttemplate.Name})
	tservers, err := listers.TSLister.TServers(namespace).List(labels.NewSelector().Add(*requirement))
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "tservers", err.Error())
	}
	if tservers != nil && len(tservers) != 0 {
		return fmt.Errorf("cannot delete ttemplate %s/%s because it is reference by some tserver", namespace, ttemplate.Name)
	}

	if !listers.TTSynced() {
		return fmt.Errorf("ttemplate infomer has not finished syncing")
	}

	requirement, _ = labels.NewRequirement(tarsMeta.TTemplateParentLabel, selection.DoubleEquals, []string{ttemplate.Name})
	ttemplates, err := listers.TTLister.ByNamespace(namespace).List(labels.NewSelector().Add(*requirement))
	if err != nil {
		return fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "ttemplates", err.Error())
	}

	if ttemplates != nil && len(ttemplates) != 0 {
		return fmt.Errorf("cannot delete ttemplate %s/%s because it is reference by some ttemplate", namespace, ttemplate.Name)
	}

	return nil
}
