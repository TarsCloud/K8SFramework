package v1beta2

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	patchTypes "k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
	crdMeta "k8s.tars.io/api/meta"
	"sort"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

type TConfigReconciler struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func (r *TConfigReconciler) splitKey(key string) (namespace, event, name string) {
	v := strings.Split(key, "/")
	return v[0], v[1], v[2]
}

func (r *TConfigReconciler) splitValue(key string) (app, server, configName, podSeq string) {
	v := strings.Split(key, "/")
	return v[0], v[1], v[2], v[3]
}

func (r *TConfigReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	if resourceName != "tconfig" {
		return
	}

	tconfigMetadataObj := resourceObj.(k8sMetaV1.Object)
	namespace := tconfigMetadataObj.GetNamespace()
	var key string
	switch resourceEvent {
	case k8sWatchV1.Added:
		objLabels := tconfigMetadataObj.GetLabels()
		app, _ := objLabels[crdMeta.TServerAppLabel]
		server, _ := objLabels[crdMeta.TServerNameLabel]
		configName, _ := objLabels[crdMeta.TConfigNameLabel]
		podSeq, _ := objLabels[crdMeta.TConfigPodSeqLabel]
		key = fmt.Sprintf("%s/%s/%s.%s.%s.%s", namespace, k8sWatchV1.Added, app, server, configName, podSeq)
		key = fmt.Sprintf("%s/%s/%s", namespace, k8sWatchV1.Modified, "")
	case k8sWatchV1.Modified, k8sWatchV1.Deleted:
		key = fmt.Sprintf("%s/%s/%s", namespace, resourceEvent, "")
	}
	r.workQueue.Add(key)
	return
}

func (r *TConfigReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func NewTConfigReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TConfigReconciler {
	reconciler := &TConfigReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconciler)
	return reconciler
}

func (r *TConfigReconciler) processItem() bool {

	obj, shutdown := r.workQueue.Get()

	if shutdown {
		return false
	}

	defer r.workQueue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		utilRuntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		r.workQueue.Forget(obj)
		return true
	}

	res := r.reconcile(key)
	switch res {
	case reconcile.AllOk:
		r.workQueue.Forget(obj)
		return true
	case reconcile.RateLimit:
		r.workQueue.AddRateLimited(obj)
		return true
	case reconcile.FatalError:
		r.workQueue.ShutDown()
		return false
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TConfigReconciler) reconcile(key string) reconcile.Result {
	namespace, event, value := r.splitKey(key)

	switch event {
	case string(k8sWatchV1.Added):
		app, server, configName, podSeq := r.splitValue(value)
		appRequirement, _ := labels.NewRequirement(crdMeta.TServerAppLabel, selection.DoubleEquals, []string{app})
		serverRequirement, _ := labels.NewRequirement(crdMeta.TServerNameLabel, selection.DoubleEquals, []string{server})
		configNameRequirement, _ := labels.NewRequirement(crdMeta.TConfigNameLabel, selection.DoubleEquals, []string{configName})
		podSeqRequirement, _ := labels.NewRequirement(crdMeta.TConfigPodSeqLabel, selection.DoubleEquals, []string{podSeq})
		activatedRequirement, _ := labels.NewRequirement(crdMeta.TConfigActivatedLabel, selection.DoubleEquals, []string{"false"})
		deletingRequirement, _ := labels.NewRequirement(crdMeta.TConfigDeletingLabel, selection.DoesNotExist, nil)

		labelSelector := labels.NewSelector().
			Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement).
			Add(*podSeqRequirement).Add(*activatedRequirement).Add(*deletingRequirement)

		tconfigs, err := r.informers.TConfigInformer.Lister().ByNamespace(namespace).List(labelSelector)
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceSelectorError, namespace, "tconfig", err.Error()))
			return reconcile.RateLimit
		}

		maxTConfigHistory := crdMeta.DefaultMaxTConfigHistory
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
				err = fmt.Errorf(crdMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels value is nil", "tconfig", namespace, obj.GetName()))
				utilRuntime.HandleError(err)
				continue
			}

			version, ok := objLabels[crdMeta.TConfigVersionLabel]
			if !ok || version == "" {
				err = fmt.Errorf(crdMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels[%s] value is nil", "tconfig", namespace, obj.GetName(), crdMeta.TConfigVersionLabel))
				utilRuntime.HandleError(err)
				continue
			}
			versions = append(versions, version)
		}
		sort.Strings(versions)
		compareVersion := versions[maxTConfigHistory-len(tconfigs)]
		compareRequirement, _ := labels.NewRequirement(crdMeta.TConfigVersionLabel, selection.LessThan, []string{compareVersion})
		labelSelector = labelSelector.Add(*compareRequirement)
		err = r.clients.CrdClient.CrdV1beta2().TConfigs(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
			LabelSelector: labelSelector.String(),
		})

		if err != nil {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceDeleteCollectionError, "tconfig", labelSelector.String(), err.Error()))
			return reconcile.RateLimit
		}

	case string(k8sWatchV1.Modified):
		deactivateRequirement, _ := labels.NewRequirement(crdMeta.TConfigDeactivateLabel, selection.Exists, nil)
		deactivateLabelSelector := labels.NewSelector().Add(*deactivateRequirement)
		tconfigs, err := r.informers.TConfigInformer.Lister().ByNamespace(namespace).List(deactivateLabelSelector)
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceSelectorError, namespace, "tconfig", err.Error()))
			return reconcile.RateLimit
		}

		jsonPatch := crdMeta.JsonPatch{
			{
				OP:   crdMeta.JsonPatchRemove,
				Path: "/metadata/labels/tars.io~1Deactivate",
			},
			{
				OP:    crdMeta.JsonPatchAdd,
				Path:  "/metadata/labels/tars.io~1Activated",
				Value: "false",
			},
			{
				OP:    crdMeta.JsonPatchReplace,
				Path:  "/activated",
				Value: false,
			},
		}
		patchContent, _ := json.Marshal(jsonPatch)

		for _, tconfig := range tconfigs {
			v := tconfig.(k8sMetaV1.Object)
			name := v.GetName()
			_, err = r.clients.CrdClient.CrdV1beta2().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
			if err != nil {
				utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourcePatchError, "tconfig", namespace, name, err.Error()))
				return reconcile.RateLimit
			}
		}
	case string(k8sWatchV1.Deleted):
		deletingRequirement, _ := labels.NewRequirement(crdMeta.TConfigDeletingLabel, selection.Exists, nil)
		deletingLabelSelector := labels.NewSelector().Add(*deletingRequirement)
		err := r.clients.CrdClient.CrdV1beta2().TConfigs(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
			LabelSelector: deletingLabelSelector.String(),
		})

		if err != nil {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceDeleteCollectionError, "tconfig", deletingLabelSelector.String(), err.Error()))
			return reconcile.RateLimit
		}
	}

	return reconcile.AllOk
}
