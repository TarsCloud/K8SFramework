package v1beta2

import (
	"context"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	tarsV1beta2 "k8s.tars.io/apis/tars/v1beta2"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"strings"
	"tarswebhook/webhook/lister"

	"tarswebhook/webhook/validating"
	"time"
)

func prepareActiveTConfig(newTConfig *tarsV1beta2.TConfig, listers *lister.Listers) error {
	if !listers.TCSynced() {
		return fmt.Errorf("tconfig infomer has not finished syncing")
	}

	namespace := newTConfig.Namespace
	appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{newTConfig.App})
	serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{newTConfig.Server})
	configNameRequirement, _ := labels.NewRequirement(tarsMeta.TConfigNameLabel, selection.DoubleEquals, []string{newTConfig.ConfigName})
	podSeqRequirement, _ := labels.NewRequirement(tarsMeta.TConfigPodSeqLabel, selection.DoubleEquals, []string{newTConfig.PodSeq})
	activateRequirement, _ := labels.NewRequirement(tarsMeta.TConfigActivatedLabel, selection.DoubleEquals, []string{"true"})

	labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement).Add(*podSeqRequirement).Add(*activateRequirement)

	tconfigs, err := listers.TCLister.ByNamespace(namespace).List(labelSelector)
	if err != nil {
		err = fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "tconfig", err.Error())
		utilRuntime.HandleError(err)
		return err
	}

	var names []string
	for _, tconfig := range tconfigs {
		obj := tconfig.(k8sMetaV1.Object)
		name := obj.GetName()
		if name == newTConfig.Name {
			continue
		}
		names = append(names, name)
	}

	counts := len(names)
	if counts == 0 {
		return nil
	}

	if counts != 1 {
		err = fmt.Errorf("get unexpected activated tconfigs(%s/%s-%s/%s:%s) counts(%d)", namespace, newTConfig.App, newTConfig.Server, newTConfig.ConfigName, newTConfig.PodSeq, counts)
		utilRuntime.HandleError(err)
		return err
	}

	jsonPatch := tarsTool.JsonPatch{
		tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Deactivate",
			Value: "Deactivating",
		},
	}
	patchContent, _ := json.Marshal(jsonPatch)
	_, err = tarsRuntime.Clients.CrdClient.TarsV1beta2().TConfigs(namespace).Patch(context.TODO(), names[0], patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
	if err != nil {
		err = fmt.Errorf(tarsMeta.ResourcePatchError, "tconfig", namespace, names[0], err.Error())
		utilRuntime.HandleError(err)
		return err
	}
	return nil
}

func prepareDeleteTConfig(tconfig *tarsV1beta2.TConfig, listers *lister.Listers) error {
	if !listers.TCSynced() {
		return fmt.Errorf("tconfig infomer has not finished syncing")
	}

	appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tconfig.App})
	serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{tconfig.Server})
	configNameRequirement, _ := labels.NewRequirement(tarsMeta.TConfigNameLabel, selection.DoubleEquals, []string{tconfig.ConfigName})

	labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement)
	if tconfig.PodSeq != "m" {
		podSeqRequirement, _ := labels.NewRequirement(tarsMeta.TConfigPodSeqLabel, selection.DoubleEquals, []string{tconfig.PodSeq})
		labelSelector = labelSelector.Add(*podSeqRequirement)
	}

	namespace := tconfig.Namespace
	tconfigRuntimeObjects, err := listers.TCLister.ByNamespace(namespace).List(labelSelector)

	if err != nil && !errors.IsNotFound(err) {
		err = fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "tconfig", err.Error())
		utilRuntime.HandleError(err)
		return err
	}

	jsonPatch := tarsTool.JsonPatch{
		tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Deleting",
			Value: "Deleting",
		},
	}
	patchContent, _ := json.Marshal(jsonPatch)

	if tconfig.PodSeq != "m" {
		for _, tconfigRuntimeObj := range tconfigRuntimeObjects {
			tconfigObj := tconfigRuntimeObj.(k8sMetaV1.Object)
			name := tconfigObj.GetName()

			if name == tconfig.Name {
				continue
			}

			_, err = tarsRuntime.Clients.CrdClient.TarsV1beta2().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
			if err != nil {
				err = fmt.Errorf(tarsMeta.ResourcePatchError, "tconfig", namespace, name, err.Error())
				utilRuntime.HandleError(err)
				return err
			}
		}
		return nil
	}

	var willMarkDeletingTConfig []string

	for _, tconfigRuntimeObj := range tconfigRuntimeObjects {
		tconfigObj := tconfigRuntimeObj.(k8sMetaV1.Object)
		name := tconfigObj.GetName()

		if name == tconfig.Name {
			continue
		}

		tconfigObjLabels := tconfigObj.GetLabels()
		if tconfigObjLabels == nil {
			err = fmt.Errorf(tarsMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels value is nil", "tconfig", namespace, tconfigObj.GetName()))
			utilRuntime.HandleError(err)
			return err
		}

		podSeq, ok := tconfigObjLabels[tarsMeta.TConfigPodSeqLabel]
		if !ok || len(podSeq) == 0 {
			err = fmt.Errorf(tarsMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels[%s] value is nil", "tconfig", namespace, tconfigObj.GetName(), tarsMeta.TConfigPodSeqLabel))
			utilRuntime.HandleError(err)
			return err
		}

		if podSeq != "m" {
			err = fmt.Errorf("cannot delete tconfig %s/%s because it is reference by anther tconfig", namespace, tconfig.Name)
			utilRuntime.HandleError(err)
			return err
		}

		willMarkDeletingTConfig = append(willMarkDeletingTConfig, name)
	}

	for _, name := range willMarkDeletingTConfig {
		_, err = tarsRuntime.Clients.CrdClient.TarsV1beta2().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
		if err != nil {
			err = fmt.Errorf(tarsMeta.ResourcePatchError, "tconfig", namespace, name, err.Error())
			utilRuntime.HandleError(err)
			return err
		}
	}

	return nil
}

func validCreateTConfig(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	newTConfig := &tarsV1beta2.TConfig{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTConfig)

	if _, ok := newTConfig.Labels[tarsMeta.TConfigDeactivateLabel]; ok {
		return fmt.Errorf("can not set label [%s] when create", tarsMeta.TConfigDeactivateLabel)
	}

	if _, ok := newTConfig.Labels[tarsMeta.TConfigDeletingLabel]; ok {
		return fmt.Errorf("can not set label [%s] when create", tarsMeta.TConfigDeletingLabel)
	}

	if len(newTConfig.Server) == 0 && newTConfig.PodSeq != "m" {
		return fmt.Errorf("app level tconfig does not support master/slave")
	}

	if newTConfig.PodSeq != "m" {
		appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{newTConfig.App})
		serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{newTConfig.Server})
		configNameRequirement, _ := labels.NewRequirement(tarsMeta.TConfigNameLabel, selection.DoubleEquals, []string{newTConfig.ConfigName})
		podSeqRequirement, _ := labels.NewRequirement(tarsMeta.TConfigPodSeqLabel, selection.DoubleEquals, []string{"m"})
		activatedRequirement, _ := labels.NewRequirement(tarsMeta.TConfigActivatedLabel, selection.DoubleEquals, []string{"true"})
		deactivatingRequirement, _ := labels.NewRequirement(tarsMeta.TConfigDeactivateLabel, selection.DoesNotExist, []string{})
		deletingRequirement, _ := labels.NewRequirement(tarsMeta.TConfigDeletingLabel, selection.DoesNotExist, []string{})
		labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement).Add(*podSeqRequirement).
			Add(*activatedRequirement).Add(*deactivatingRequirement).Add(*deletingRequirement)
		namespace := newTConfig.Namespace
		tconfigRuntimeObjects, err := listers.TCLister.ByNamespace(namespace).List(labelSelector)

		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("no activated master tconfig found")
			}
			err = fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "tconfig", err.Error())
			utilRuntime.HandleError(err)
			return err
		}

		if len(tconfigRuntimeObjects) == 0 {
			return fmt.Errorf("no activated master tconfig found")
		}
	}

	if newTConfig.Activated {
		return prepareActiveTConfig(newTConfig, listers)
	}
	return nil
}

func validUpdateTConfig(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	newTConfig := &tarsV1beta2.TConfig{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTConfig)

	oldTConfig := &tarsV1beta2.TConfig{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTConfig)

	if _, ok := oldTConfig.Labels[tarsMeta.TConfigDeletingLabel]; ok {
		return fmt.Errorf("can not update deleting tconfig")
	}

	if newTConfig.App != oldTConfig.App {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "tconfig", "/app")
	}
	if newTConfig.Server != oldTConfig.Server {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "tconfig", "/server")
	}
	if newTConfig.PodSeq != oldTConfig.PodSeq {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "tconfig", "/podSeq")
	}
	if newTConfig.ConfigName != oldTConfig.ConfigName {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "tconfig", "/configName")
	}
	if newTConfig.ConfigContent != oldTConfig.ConfigContent {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "tconfig", "/configContent")
	}
	if newTConfig.Version != oldTConfig.Version {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "tconfig", "/version")
	}
	if newTConfig.UpdateTime != oldTConfig.UpdateTime {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "TConfig", "/updateTime")
	}
	if newTConfig.UpdatePerson != oldTConfig.UpdatePerson {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "TConfig", "/updatePerson")
	}
	if newTConfig.UpdateReason != oldTConfig.UpdateReason {
		return fmt.Errorf(tarsMeta.FiledImmutableError, "TConfig", "/updateReason")
	}

	if !newTConfig.Activated && oldTConfig.Activated {
		return fmt.Errorf("only use authorized account can update /activated from true to false")
	}

	if !oldTConfig.Activated && newTConfig.Activated {
		return prepareActiveTConfig(newTConfig, listers)
	}

	return nil
}

func validDeleteTConfig(listers *lister.Listers, view *k8sAdmissionV1.AdmissionReview) error {
	requestServiceAccount := view.Request.UserInfo.Username
	controllerUserName := tarsMeta.DefaultControllerServiceAccount
	if requestServiceAccount == controllerUserName {
		return nil
	}

	if strings.HasPrefix(requestServiceAccount, tarsMeta.KubernetesSystemAccountPrefix) {
		return nil
	}

	tconfig := &tarsV1beta2.TConfig{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, tconfig)

	if !tconfig.Activated {
		return nil
	}

	if !tconfig.CreationTimestamp.Add(48 * time.Second).Before(time.Now()) {
		return fmt.Errorf("tconfig during deletion guard time(48s)")
	}

	if _, ok := tconfig.Labels[tarsMeta.TConfigDeactivateLabel]; ok {
		return nil
	}

	if _, ok := tconfig.Labels[tarsMeta.TConfigDeletingLabel]; ok {
		return nil
	}

	return prepareDeleteTConfig(tconfig, listers)
}

func init() {
	gvr := tarsV1beta2.SchemeGroupVersion.WithResource("tconfigs")
	validating.Registry(k8sAdmissionV1.Create, &gvr, validCreateTConfig)
	validating.Registry(k8sAdmissionV1.Update, &gvr, validUpdateTConfig)
	validating.Registry(k8sAdmissionV1.Delete, &gvr, validDeleteTConfig)
}
