package v1beta3

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"sort"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/util"
	"time"
)

type TConfigReconciler struct {
	clients     *util.Clients
	tcLister    cache.GenericLister
	threads     int
	addQueue    workqueue.RateLimitingInterface
	modifyQueue workqueue.RateLimitingInterface
	deleteQueue workqueue.RateLimitingInterface
	synced      []cache.InformerSynced
}

func (r *TConfigReconciler) splitAddKey(key string) (namespace, app, server, configName, podSeq string) {
	v := strings.Split(key, "/")
	return v[0], v[1], v[2], v[3], v[4]
}

func (r *TConfigReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	if resourceKind == tarsMeta.TConfigKind {
		tconfigMetadataObj := resourceObj.(k8sMetaV1.Object)
		namespace := tconfigMetadataObj.GetNamespace()
		switch resourceEvent {
		case k8sWatchV1.Added:
			r.modifyQueue.Add(namespace)
			objLabels := tconfigMetadataObj.GetLabels()
			app, _ := objLabels[tarsMeta.TServerAppLabel]
			server, _ := objLabels[tarsMeta.TServerNameLabel]
			configName, _ := objLabels[tarsMeta.TConfigNameLabel]
			podSeq, _ := objLabels[tarsMeta.TConfigPodSeqLabel]
			key := fmt.Sprintf("%s/%s/%s/%s/%s", namespace, app, server, configName, podSeq)
			r.addQueue.Add(key)
		case k8sWatchV1.Modified:
			r.modifyQueue.Add(namespace)
			r.deleteQueue.Add(namespace)
		case k8sWatchV1.Deleted:
			r.deleteQueue.Add(namespace)
		}
	}
}

func (r *TConfigReconciler) reconcileAdd(key string) controller.Result {
	namespace, app, server, configName, podSeq := r.splitAddKey(key)
	appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{app})
	serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{server})
	configNameRequirement, _ := labels.NewRequirement(tarsMeta.TConfigNameLabel, selection.DoubleEquals, []string{configName})
	podSeqRequirement, _ := labels.NewRequirement(tarsMeta.TConfigPodSeqLabel, selection.DoubleEquals, []string{podSeq})
	activatedRequirement, _ := labels.NewRequirement(tarsMeta.TConfigActivatedLabel, selection.DoubleEquals, []string{"false"})
	deletingRequirement, _ := labels.NewRequirement(tarsMeta.TConfigDeletingLabel, selection.DoesNotExist, nil)

	labelSelector := labels.NewSelector().
		Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement).
		Add(*podSeqRequirement).Add(*activatedRequirement).Add(*deletingRequirement)

	tconfigs, err := r.tcLister.ByNamespace(namespace).List(labelSelector)
	if err != nil && !errors.IsNotFound(err) {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "tconfig", err.Error()))
		return controller.Retry
	}

	maxTConfigHistory := tarsMeta.DefaultMaxTConfigHistory
	if tfc := util.GetTFrameworkConfig(namespace); tfc != nil {
		maxTConfigHistory = tfc.RecordLimit.TConfigHistory
	}

	if len(tconfigs) <= maxTConfigHistory {
		return controller.Done
	}

	var versions []string
	for _, tconfig := range tconfigs {
		obj := tconfig.(k8sMetaV1.Object)
		objLabels := obj.GetLabels()
		if objLabels == nil {
			err = fmt.Errorf(tarsMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels value is nil", "tconfig", namespace, obj.GetName()))
			utilRuntime.HandleError(err)
			continue
		}

		version, ok := objLabels[tarsMeta.TConfigVersionLabel]
		if !ok || version == "" {
			err = fmt.Errorf(tarsMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels[%s] value is nil", "tconfig", namespace, obj.GetName(), tarsMeta.TConfigVersionLabel))
			utilRuntime.HandleError(err)
			continue
		}
		versions = append(versions, version)
	}
	sort.Strings(versions)
	compareVersion := versions[maxTConfigHistory-len(tconfigs)]
	compareRequirement, _ := labels.NewRequirement(tarsMeta.TConfigVersionLabel, selection.LessThan, []string{compareVersion})
	labelSelector = labelSelector.Add(*compareRequirement)
	err = r.clients.CrdClient.CrdV1beta3().TConfigs(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteCollectionError, "tconfig", labelSelector.String(), err.Error()))
		return controller.Retry
	}

	return controller.Done
}

func (r *TConfigReconciler) reconcileModify(key string) controller.Result {
	namespace := key
	deactivateRequirement, _ := labels.NewRequirement(tarsMeta.TConfigDeactivateLabel, selection.Exists, nil)
	deactivateLabelSelector := labels.NewSelector().Add(*deactivateRequirement)
	tconfigs, err := r.tcLister.ByNamespace(namespace).List(deactivateLabelSelector)
	if err != nil && !errors.IsNotFound(err) {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "tconfig", err.Error()))
		return controller.Retry
	}

	jsonPatch := tarsMeta.JsonPatch{
		{
			OP:   tarsMeta.JsonPatchRemove,
			Path: "/metadata/labels/tars.io~1Deactivate",
		},
		{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Activated",
			Value: "false",
		},
		{
			OP:    tarsMeta.JsonPatchReplace,
			Path:  "/activated",
			Value: false,
		},
	}
	patchContent, _ := json.Marshal(jsonPatch)
	retry := false
	for _, tconfig := range tconfigs {
		v := tconfig.(k8sMetaV1.Object)
		name := v.GetName()
		_, err = r.clients.CrdClient.CrdV1beta3().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourcePatchError, "tconfig", namespace, name, err.Error()))
			retry = true
			continue
		}
	}
	if retry {
		return controller.Retry
	}
	return controller.Done
}

func (r *TConfigReconciler) reconcileDelete(key string) controller.Result {
	namespace := key
	deletingRequirement, _ := labels.NewRequirement(tarsMeta.TConfigDeletingLabel, selection.Exists, nil)
	deletingLabelSelector := labels.NewSelector().Add(*deletingRequirement)
	err := r.clients.CrdClient.CrdV1beta3().TConfigs(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
		LabelSelector: deletingLabelSelector.String(),
	})
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteCollectionError, "tconfig", deletingLabelSelector.String(), err.Error()))
		return controller.Retry
	}
	return controller.Done
}

func (r *TConfigReconciler) StartController(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.addQueue.ShutDown()
	defer r.modifyQueue.ShutDown()
	defer r.deleteQueue.ShutDown()

	if !cache.WaitForNamedCacheSync("tconfig controller", stopCh, r.synced...) {
		return
	}

	for i := 0; i < r.threads; i++ {
		go wait.Until(func() { r.processItem(r.addQueue, r.reconcileAdd) }, time.Second, stopCh)
		go wait.Until(func() { r.processItem(r.modifyQueue, r.reconcileModify) }, time.Second, stopCh)
		go wait.Until(func() { r.processItem(r.deleteQueue, r.reconcileDelete) }, time.Second, stopCh)
	}

	<-stopCh
}

func (r *TConfigReconciler) processItem(queue workqueue.RateLimitingInterface, reconciler func(key string) controller.Result) bool {
	obj, shutdown := queue.Get()

	if shutdown {
		return false
	}

	defer queue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		utilRuntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		queue.Forget(obj)
		return true
	}

	res := reconciler(key)
	switch res {
	case controller.Done:
		queue.Forget(obj)
		return true
	case controller.Retry:
		queue.AddRateLimited(obj)
		return true
	case controller.FatalError:
		queue.ShutDown()
		return false
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func NewTConfigController(clients *util.Clients, factories *util.InformerFactories, threads int) *TConfigReconciler {
	tcInformer := factories.MetadataInformerFactor.ForResource(tarsCrdV1beta3.SchemeGroupVersion.WithResource("tconfigs"))
	c := &TConfigReconciler{
		clients:     clients,
		tcLister:    tcInformer.Lister(),
		threads:     threads,
		addQueue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		modifyQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		deleteQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:      []cache.InformerSynced{tcInformer.Informer().HasSynced},
	}
	controller.SetInformerHandlerEvent(tarsMeta.TConfigKind, tcInformer.Informer(), c)
	return c
}
