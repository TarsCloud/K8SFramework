package v1beta3

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMetaV1beta3 "k8s.tars.io/meta/v1beta3"
	"strings"
	"tarscontroller/controller"
)

func validTTemplate(newTTemplate *tarsCrdV1beta3.TTemplate, oldTTemplate *tarsCrdV1beta3.TTemplate, clients *controller.Clients, informers *controller.Informers) error {

	parentName := newTTemplate.Spec.Parent
	if parentName == "" {
		return fmt.Errorf(tarsMetaV1beta3.ResourceInvalidError, "ttemplate", "value of filed \".spec.parent\" should not empty ")
	}

	if newTTemplate.Name == newTTemplate.Spec.Parent {
		return nil
	}

	namespace := newTTemplate.Namespace
	_, err := informers.TTemplateInformer.Lister().ByNamespace(namespace).Get(parentName)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf(tarsMetaV1beta3.ResourceGetError, "ttemplate", namespace, parentName, err.Error())
		}
		return fmt.Errorf(tarsMetaV1beta3.ResourceNotExistError, "ttemplate", namespace, parentName)
	}

	return nil
}

func validCreateTTemplate(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)
	return validTTemplate(newTTemplate, nil, clients, informers)
}

func validUpdateTTemplate(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)

	oldTTemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTemplate)

	return validTTemplate(newTTemplate, oldTTemplate, clients, informers)
}

func validDeleteTTemplate(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	username := view.Request.UserInfo.Username
	controllerUserName := controller.GetControllerUsername()

	if controllerUserName == username || controllerUserName == tarsMetaV1beta3.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	if strings.HasPrefix(username, tarsMetaV1beta3.KubernetesSystemAccountPrefix) {
		return nil
	}

	ttemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttemplate)
	namespace := ttemplate.Namespace

	requirement, _ := labels.NewRequirement(tarsMetaV1beta3.TemplateLabel, selection.DoubleEquals, []string{ttemplate.Name})
	tservers, err := informers.TServerInformer.Lister().TServers(namespace).List(labels.NewSelector().Add(*requirement))
	if err != nil {
		return fmt.Errorf(tarsMetaV1beta3.ResourceSelectorError, namespace, "tservers", err.Error())
	}
	if tservers != nil && len(tservers) != 0 {
		return fmt.Errorf("cannot delete ttemplate %s/%s because it is reference by some tserver", namespace, ttemplate.Name)
	}

	requirement, _ = labels.NewRequirement(tarsMetaV1beta3.ParentLabel, selection.DoubleEquals, []string{ttemplate.Name})
	ttemplates, err := informers.TTemplateInformer.Lister().ByNamespace(namespace).List(labels.NewSelector().Add(*requirement))
	if err != nil {
		return fmt.Errorf(tarsMetaV1beta3.ResourceSelectorError, namespace, "ttemplates", err.Error())
	}
	if ttemplates != nil && len(ttemplates) != 0 {
		return fmt.Errorf("cannot delete ttemplate %s/%s because it is reference by some ttemplate", namespace, ttemplate.Name)
	}

	return nil
}
