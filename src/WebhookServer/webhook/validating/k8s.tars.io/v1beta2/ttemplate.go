package v1beta2

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	tarsAppsV1beta2 "k8s.tars.io/apps/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"strings"
	"tarswebhook/webhook/informer"
)

func validTTemplate(newTTemplate *tarsAppsV1beta2.TTemplate, oldTTemplate *tarsAppsV1beta2.TTemplate, listers *informer.Listers) error {

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

func validCreateTTemplate(informers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &tarsAppsV1beta2.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)
	return validTTemplate(newTTemplate, nil, informers)
}

func validUpdateTTemplate(informers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &tarsAppsV1beta2.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)

	oldTTemplate := &tarsAppsV1beta2.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTemplate)

	return validTTemplate(newTTemplate, oldTTemplate, informers)
}

func validDeleteTTemplate(listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	username := view.Request.UserInfo.Username
	controllerUserName := tarsRuntime.Username

	if controllerUserName == username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	if strings.HasPrefix(username, tarsMeta.KubernetesSystemAccountPrefix) {
		return nil
	}

	ttemplate := &tarsAppsV1beta2.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttemplate)
	namespace := ttemplate.Namespace

	{
		if !listers.TTSynced() {
			return fmt.Errorf("ttemplate infomer has not finished syncing")
		}

		requirement, _ := labels.NewRequirement(tarsMeta.TTemplateParentLabel, selection.DoubleEquals, []string{ttemplate.Name})
		ttemplates, err := listers.TTLister.ByNamespace(namespace).List(labels.NewSelector().Add(*requirement))
		if err != nil {
			return fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "ttemplates", err.Error())
		}
		if ttemplates != nil && len(ttemplates) != 0 {
			return fmt.Errorf("cannot delete ttemplate %s/%s because it is reference by some ttemplate", namespace, ttemplate.Name)
		}
	}

	{
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
	}

	return nil
}
