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
	"k8s.io/klog/v2"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"sort"
	"strings"
	"tarscontroller/controller"
	"time"
)

type TConfigReconciler struct {
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
			objLabels := tconfigMetadataObj.GetLabels()
			app, _ := objLabels[tarsMeta.TServerAppLabel]
			server, _ := objLabels[tarsMeta.TServerNameLabel]
			configName, _ := objLabels[tarsMeta.TConfigNameLabel]
			podSeq, _ := objLabels[tarsMeta.TConfigPodSeqLabel]
			key := fmt.Sprintf("%s/%s/%s/%s/%s", namespace, app, server, configName, podSeq)
			r.addQueue.Add(key)
			r.modifyQueue.Add(namespace)
		case k8sWatchV1.Modified:
			r.modifyQueue.Add(namespace)
			r.deleteQueue.Add(namespace)
		case k8sWatchV1.Deleted:
			r.deleteQueue.Add(namespace)
		}
	}
}

func (r *TConfigReconciler) reconcileAdded(key string) controller.Result {
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
		klog.Errorf(tarsMeta.ResourceSelectorError, namespace, "tconfig", err.Error())
		return controller.Retry
	}

	maxTConfigHistory := tarsMeta.DefaultMaxTConfigHistory
	if tfc := tarsRuntime.TFCConfig.GetTFrameworkConfig(namespace); tfc != nil {
		maxTConfigHistory = tfc.RecordLimit.TConfigHistory
	}

	if len(tconfigs) <= maxTConfigHistory {
		return controller.Done
	}

	var versions []string
	versionNameMap := map[string]string{}
	for _, tconfig := range tconfigs {
		obj := tconfig.(k8sMetaV1.Object)
		objLabels := obj.GetLabels()
		if objLabels == nil {
			klog.Errorf(tarsMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels value is nil", "tconfig", namespace, obj.GetName()))
			continue
		}

		version, ok := objLabels[tarsMeta.TConfigVersionLabel]
		if !ok || version == "" {
			klog.Errorf(tarsMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels[%s] value is nil", "tconfig", namespace, obj.GetName(), tarsMeta.TConfigVersionLabel))
			continue
		}
		versions = append(versions, version)
	}
	sort.Strings(versions)
	name := versionNameMap[versions[0]]
	err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TConfigs(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf(tarsMeta.ResourceDeleteError, "tconfig", namespace, name, err.Error())
		return controller.Retry
	}
	return controller.Done
}

func (r *TConfigReconciler) reconcileModified(key string) controller.Result {
	namespace := key
	deactivateRequirement, _ := labels.NewRequirement(tarsMeta.TConfigDeactivateLabel, selection.Exists, nil)
	deactivateLabelSelector := labels.NewSelector().Add(*deactivateRequirement)
	tconfigs, err := r.tcLister.ByNamespace(namespace).List(deactivateLabelSelector)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf(tarsMeta.ResourceSelectorError, namespace, "tconfig", err.Error())
		return controller.Retry
	}

	jsonPatch := tarsTool.JsonPatch{
		{
			OP:   tarsTool.JsonPatchRemove,
			Path: "/metadata/labels/tars.io~1Deactivate",
		},
		{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Activated",
			Value: "false",
		},
		{
			OP:    tarsTool.JsonPatchReplace,
			Path:  "/activated",
			Value: false,
		},
	}
	patchContent, _ := json.Marshal(jsonPatch)
	retry := false
	for _, tconfig := range tconfigs {
		v := tconfig.(k8sMetaV1.Object)
		name := v.GetName()
		_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
		if err != nil {
			klog.Errorf(tarsMeta.ResourcePatchError, "tconfig", namespace, name, err.Error())
			retry = true
			continue
		}
	}
	if retry {
		return controller.Retry
	}
	return controller.Done
}

func (r *TConfigReconciler) reconcileDeleted(key string) controller.Result {
	namespace := key
	deletingRequirement, _ := labels.NewRequirement(tarsMeta.TConfigDeletingLabel, selection.Exists, nil)
	deletingLabelSelector := labels.NewSelector().Add(*deletingRequirement)
	err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TConfigs(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
		LabelSelector: deletingLabelSelector.String(),
	})
	if err != nil {
		klog.Errorf(tarsMeta.ResourceDeleteCollectionError, "tconfig", deletingLabelSelector.String(), err.Error())
		return controller.Retry
	}
	return controller.Done
}

func (r *TConfigReconciler) Run(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.addQueue.ShutDown()
	defer r.modifyQueue.ShutDown()
	defer r.deleteQueue.ShutDown()

	if !cache.WaitForNamedCacheSync("tconfig controller", stopCh, r.synced...) {
		return
	}

	for i := 0; i < r.threads; i++ {
		go wait.Until(func() { r.processItem(r.addQueue, r.reconcileAdded) }, time.Second, stopCh)
	}

	for i := 0; i < 3; i++ {
		go wait.Until(func() { r.processItem(r.modifyQueue, r.reconcileModified) }, time.Second, stopCh)
	}

	for i := 0; i < 1; i++ {
		go wait.Until(func() { r.processItem(r.deleteQueue, r.reconcileDeleted) }, time.Second, stopCh)
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
		klog.Errorf("expected string in workqueue but got %#v", obj)
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
		klog.Errorf("should not reach place")
		return false
	}
}

func NewTConfigController(threads int) *TConfigReconciler {
	tcInformer := tarsRuntime.Factories.MetadataInformerFactor.ForResource(tarsV1beta3.SchemeGroupVersion.WithResource("tconfigs"))
	c := &TConfigReconciler{
		tcLister:    tcInformer.Lister(),
		threads:     threads,
		addQueue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		modifyQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		deleteQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:      []cache.InformerSynced{tcInformer.Informer().HasSynced},
	}
	controller.RegistryInformerEventHandle(tarsMeta.TConfigKind, tcInformer.Informer(), c)
	return c
}
