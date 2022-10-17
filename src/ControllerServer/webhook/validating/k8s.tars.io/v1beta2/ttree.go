package v1beta2

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	tarsCrdV1beta2 "k8s.tars.io/crd/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarscontroller/controller"
)

func validTTree(newTTree *tarsCrdV1beta2.TTree, oldTTree *tarsCrdV1beta2.TTree, clients *controller.Clients, informers *controller.Informers) error {
	namespace := newTTree.Namespace

	businessMap := make(map[string]interface{}, len(newTTree.Businesses))
	for _, business := range newTTree.Businesses {
		if _, ok := businessMap[business.Name]; ok {
			return fmt.Errorf(tarsMeta.ResourceInvalidError, "ttree", fmt.Sprintf("duplicate business name : %s", business.Name))
		}
		businessMap[business.Name] = nil
	}

	appMap := make(map[string]interface{}, len(newTTree.Apps))
	for _, app := range newTTree.Apps {
		if _, ok := appMap[app.Name]; ok {
			return fmt.Errorf(tarsMeta.ResourceInvalidError, "ttree", fmt.Sprintf("duplicate app name : %s", app.Name))
		}
		if app.BusinessRef != "" {
			if _, ok := businessMap[app.BusinessRef]; !ok {
				return fmt.Errorf(tarsMeta.ResourceInvalidError, "ttree", fmt.Sprintf("business/%s not exist", app.BusinessRef))
			}
		}
		appMap[app.Name] = nil
	}

	if oldTTree == nil {
		return nil
	}

	for i := range oldTTree.Apps {
		appName := oldTTree.Apps[i].Name
		if _, ok := appMap[appName]; !ok {
			requirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{appName})
			tservers, err := informers.TServerInformer.Lister().TServers(namespace).List(labels.NewSelector().Add(*requirement))
			if err != nil {
				utilRuntime.HandleError(err)
				return err
			}
			if tservers != nil && len(tservers) != 0 {
				return fmt.Errorf(tarsMeta.ResourceInvalidError, "ttree", fmt.Sprintf("cannot delete ttree/apps[%s] because it is reference by some tserver", appName))
			}
		}
	}
	return nil
}

func validCreateTTree(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTree := &tarsCrdV1beta2.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTree)

	if newTTree.Name != tarsMeta.FixedTTreeResourceName {
		return fmt.Errorf("create ttree operation is defined")
	}

	namespace := newTTree.Namespace

	_, err := informers.TTreeInformer.Lister().TTrees(namespace).Get(tarsMeta.FixedTTreeResourceName)
	if err == nil {
		return fmt.Errorf("create ttree operation is defined")
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("create ttree operation is defined")
	}

	return validTTree(newTTree, nil, clients, informers)
}

func validUpdateTTree(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	ttree := &tarsCrdV1beta2.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, ttree)

	oldTTree := &tarsCrdV1beta2.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTree)

	return validTTree(ttree, oldTTree, clients, informers)
}

func validDeleteTTree(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	username := view.Request.UserInfo.Username
	controllerUserName := controller.GetControllerUsername()

	if controllerUserName == username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	if strings.HasPrefix(username, tarsMeta.KubernetesSystemAccountPrefix) {
		return nil
	}

	ttree := &tarsCrdV1beta2.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttree)

	if ttree.Name == tarsMeta.FixedTTreeResourceName {
		return fmt.Errorf("delete ttree operation is defined")
	}
	return nil
}
