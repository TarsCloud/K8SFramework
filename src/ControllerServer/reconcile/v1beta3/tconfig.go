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
	"k8s.io/client-go/util/workqueue"
	tarsMetaTools "k8s.tars.io/meta/tools"
	tarsMetaV1beta3 "k8s.tars.io/meta/v1beta3"
	"sort"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

type TConfigReconciler struct {
	clients     *controller.Clients
	informers   *controller.Informers
	addQueue    workqueue.RateLimitingInterface
	modifyQueue workqueue.RateLimitingInterface
	deleteQueue workqueue.RateLimitingInterface
}

func (r *TConfigReconciler) splitAddKey(key string) (namespace, app, server, configName, podSeq string) {
	v := strings.Split(key, "/")
	return v[0], v[1], v[2], v[3], v[4]
}

func (r *TConfigReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	if resourceName != "tconfig" {
		return
	}
	tconfigMetadataObj := resourceObj.(k8sMetaV1.Object)
	namespace := tconfigMetadataObj.GetNamespace()
	var key string
	switch resourceEvent {
	case k8sWatchV1.Modified:
		r.addQueue.Add(namespace)
	case k8sWatchV1.Deleted:
		r.deleteQueue.Add(namespace)
	case k8sWatchV1.Added:
		objLabels := tconfigMetadataObj.GetLabels()
		app, _ := objLabels[tarsMetaV1beta3.TServerAppLabel]
		server, _ := objLabels[tarsMetaV1beta3.TServerNameLabel]
		configName, _ := objLabels[tarsMetaV1beta3.TConfigNameLabel]
		podSeq, _ := objLabels[tarsMetaV1beta3.TConfigPodSeqLabel]
		key = fmt.Sprintf("%s/%s/%s/%s/%s", namespace, app, server, configName, podSeq)
		r.addQueue.Add(key)
	}
	return
}

func (r *TConfigReconciler) reconcileAdd(key string) reconcile.Result {
	namespace, app, server, configName, podSeq := r.splitAddKey(key)
	appRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TServerAppLabel, selection.DoubleEquals, []string{app})
	serverRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TServerNameLabel, selection.DoubleEquals, []string{server})
	configNameRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TConfigNameLabel, selection.DoubleEquals, []string{configName})
	podSeqRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TConfigPodSeqLabel, selection.DoubleEquals, []string{podSeq})
	activatedRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TConfigActivatedLabel, selection.DoubleEquals, []string{"false"})
	deletingRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TConfigDeletingLabel, selection.DoesNotExist, nil)

	labelSelector := labels.NewSelector().
		Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement).
		Add(*podSeqRequirement).Add(*activatedRequirement).Add(*deletingRequirement)

	tconfigs, err := r.informers.TConfigInformer.Lister().ByNamespace(namespace).List(labelSelector)
	if err != nil && !errors.IsNotFound(err) {
		utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceSelectorError, namespace, "tconfig", err.Error()))
		return reconcile.RateLimit
	}

	maxTConfigHistory := tarsMetaV1beta3.DefaultMaxTConfigHistory
	if tfc := controller.GetTFrameworkConfig(namespace); tfc != nil {
		maxTConfigHistory = tfc.RecordLimit.TConfigHistory
	}

	if len(tconfigs) <= maxTConfigHistory {
		return reconcile.AllOk
	}
	var versions []string
	for _, tconfig := range tconfigs {
		obj := tconfig.(k8sMetaV1.Object)
		objLabels := obj.GetLabels()
		if objLabels == nil {
			err = fmt.Errorf(tarsMetaV1beta3.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels value is nil", "tconfig", namespace, obj.GetName()))
			utilRuntime.HandleError(err)
			continue
		}

		version, ok := objLabels[tarsMetaV1beta3.TConfigVersionLabel]
		if !ok || version == "" {
			err = fmt.Errorf(tarsMetaV1beta3.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels[%s] value is nil", "tconfig", namespace, obj.GetName(), tarsMetaV1beta3.TConfigVersionLabel))
			utilRuntime.HandleError(err)
			continue
		}
		versions = append(versions, version)
	}
	sort.Strings(versions)
	compareVersion := versions[maxTConfigHistory-len(tconfigs)]
	compareRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TConfigVersionLabel, selection.LessThan, []string{compareVersion})
	labelSelector = labelSelector.Add(*compareRequirement)
	err = r.clients.CrdClient.CrdV1beta3().TConfigs(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceDeleteCollectionError, "tconfig", labelSelector.String(), err.Error()))
		return reconcile.RateLimit
	}

	return reconcile.AllOk
}

func (r *TConfigReconciler) reconcileModify(key string) reconcile.Result {
	namespace := key
	deactivateRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TConfigDeactivateLabel, selection.Exists, nil)
	deactivateLabelSelector := labels.NewSelector().Add(*deactivateRequirement)
	tconfigs, err := r.informers.TConfigInformer.Lister().ByNamespace(namespace).List(deactivateLabelSelector)
	if err != nil && !errors.IsNotFound(err) {
		utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceSelectorError, namespace, "tconfig", err.Error()))
		return reconcile.RateLimit
	}

	jsonPatch := tarsMetaTools.JsonPatch{
		{
			OP:   tarsMetaTools.JsonPatchRemove,
			Path: "/metadata/labels/tars.io~1Deactivate",
		},
		{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Activated",
			Value: "false",
		},
		{
			OP:    tarsMetaTools.JsonPatchReplace,
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
			utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourcePatchError, "tconfig", namespace, name, err.Error()))
			retry = true
			continue
		}
	}
	if retry {
		return reconcile.RateLimit
	}
	return reconcile.AllOk
}

func (r *TConfigReconciler) reconcileDelete(key string) reconcile.Result {
	namespace := key
	deletingRequirement, _ := labels.NewRequirement(tarsMetaV1beta3.TConfigDeletingLabel, selection.Exists, nil)
	deletingLabelSelector := labels.NewSelector().Add(*deletingRequirement)
	err := r.clients.CrdClient.CrdV1beta3().TConfigs(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
		LabelSelector: deletingLabelSelector.String(),
	})
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceDeleteCollectionError, "tconfig", deletingLabelSelector.String(), err.Error()))
		return reconcile.RateLimit
	}
	return reconcile.AllOk
}

func (r *TConfigReconciler) Start(stopCh chan struct{}) {
	go wait.Until(func() { r.processItem(r.addQueue, r.reconcileAdd) }, time.Second, stopCh)
	go wait.Until(func() { r.processItem(r.modifyQueue, r.reconcileModify) }, time.Second, stopCh)
	go wait.Until(func() { r.processItem(r.deleteQueue, r.reconcileDelete) }, time.Second, stopCh)
}

func (r *TConfigReconciler) processItem(queue workqueue.RateLimitingInterface, reconciler func(key string) reconcile.Result) bool {
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
	case reconcile.AllOk:
		queue.Forget(obj)
		return true
	case reconcile.RateLimit:
		queue.AddRateLimited(obj)
		return true
	case reconcile.FatalError:
		queue.ShutDown()
		return false
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func NewTConfigReconciler(clients *controller.Clients, informers *controller.Informers, _ int) *TConfigReconciler {
	reconciler := &TConfigReconciler{
		clients:     clients,
		informers:   informers,
		addQueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "add"),
		modifyQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "modify"),
		deleteQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "delete"),
	}
	informers.Register(reconciler)
	return reconciler
}
