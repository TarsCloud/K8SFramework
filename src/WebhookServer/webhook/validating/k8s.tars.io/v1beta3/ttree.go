package v1beta3

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	tarsAppsV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"strings"
	"tarswebhook/webhook/informer"
)

func validTTree(newTTree *tarsAppsV1beta3.TTree, oldTTree *tarsAppsV1beta3.TTree, listers *informer.Listers) error {
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

	if !listers.TRSynced() {
		return fmt.Errorf("tserver infomer has not finished syncing")
	}

	for i := range oldTTree.Apps {
		appName := oldTTree.Apps[i].Name
		if _, ok := appMap[appName]; !ok {
			requirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{appName})
			tservers, err := listers.TSLister.TServers(namespace).List(labels.NewSelector().Add(*requirement))
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

func validCreateTTree(listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTree := &tarsAppsV1beta3.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTree)

	if newTTree.Name != tarsMeta.FixedTTreeResourceName {
		return fmt.Errorf("create ttree operation is defined")
	}

	if !listers.TRSynced() {
		return fmt.Errorf("ttress infomer has not finished syncing")
	}

	namespace := newTTree.Namespace
	_, err := listers.TRLister.ByNamespace(namespace).Get(tarsMeta.FixedTTreeResourceName)
	if err == nil {
		return fmt.Errorf("create ttree operation is defined")
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf(tarsMeta.ResourceGetError, namespace, "ttree", tarsMeta.FixedTTreeResourceName, err.Error())
	}

	return validTTree(newTTree, nil, listers)
}

func validUpdateTTree(informers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := tarsRuntime.Username
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	ttree := &tarsAppsV1beta3.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, ttree)

	oldTTree := &tarsAppsV1beta3.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTree)

	return validTTree(ttree, oldTTree, informers)
}

func validDeleteTTree(listers *informer.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	username := view.Request.UserInfo.Username
	controllerUserName := tarsRuntime.Username

	if controllerUserName == username || controllerUserName == tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	if strings.HasPrefix(username, tarsMeta.KubernetesSystemAccountPrefix) {
		return nil
	}

	ttree := &tarsAppsV1beta3.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttree)

	if ttree.Name == tarsMeta.FixedTTreeResourceName {
		return fmt.Errorf("delete ttree operation is defined")
	}
	return nil
}