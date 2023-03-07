package v1beta3

import (
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarswebhook/webhook/lister"

	"tarswebhook/webhook/validating"
)

func validTTree(newTTree *tarsV1beta3.TTree, oldTTree *tarsV1beta3.TTree, listers *lister.Listers) error {
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

func validCreateTTree(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTree := &tarsV1beta3.TTree{}
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

func validUpdateTTree(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	ttree := &tarsV1beta3.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, ttree)

	oldTTree := &tarsV1beta3.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTree)

	return validTTree(ttree, oldTTree, listers)
}

func validDeleteTTree(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	if strings.HasPrefix(requestServiceAccount, tarsMeta.KubernetesSystemAccountPrefix) {
		return nil
	}

	ttree := &tarsV1beta3.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttree)

	if ttree.Name == tarsMeta.FixedTTreeResourceName {
		return fmt.Errorf("delete ttree operation is defined")
	}
	return nil
}

func init() {
	gvr := tarsV1beta3.SchemeGroupVersion.WithResource("ttrees")
	validating.Registry(k8sAdmissionV1.Create, &gvr, validCreateTTree)
	validating.Registry(k8sAdmissionV1.Update, &gvr, validUpdateTTree)
	validating.Registry(k8sAdmissionV1.Delete, &gvr, validDeleteTTree)
}
