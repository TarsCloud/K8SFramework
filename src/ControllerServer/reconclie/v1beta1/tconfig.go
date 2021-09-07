package v1beta1

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	patchTypes "k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	"time"
)

type TConfigReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func (r *TConfigReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceName {
	case "tconfig":
		tconfigMetadataObj := resourceObj.(k8sMetaV1.Object)
		namespace := tconfigMetadataObj.GetNamespace()
		key := fmt.Sprintf("%s", namespace)
		r.workQueue.Add(key)
		return
	}
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

func NewTConfigReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *TConfigReconciler {
	reconcile := &TConfigReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
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
	case reconclie.AllOk:
		r.workQueue.Forget(obj)
		return true
	case reconclie.RateLimit:
		r.workQueue.AddRateLimited(obj)
		return true
	case reconclie.FatalError:
		r.workQueue.ShutDown()
		return false
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TConfigReconciler) reconcile(key string) reconclie.ReconcileResult {
	namespace := key
	deletingRequirement, _ := labels.NewRequirement(meta.TConfigDeletingLabel, "exists", nil)
	deletingLabelSelector := labels.NewSelector().Add(*deletingRequirement)
	err := r.clients.CrdClient.CrdV1beta1().TConfigs(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
		LabelSelector: deletingLabelSelector.String(),
	})
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteCollectionError, "tconfig", deletingLabelSelector.String(), err.Error()))
		return reconclie.RateLimit
	}

	deactivateRequirement, _ := labels.NewRequirement(meta.TConfigDeactivateLabel, "exists", nil)
	deactivateLabelSelector := labels.NewSelector().Add(*deactivateRequirement)
	tconfigs, err := r.informers.TConfigInformer.Lister().ByNamespace(namespace).List(deactivateLabelSelector)
	if err != nil && !errors.IsNotFound(err) {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceSelectorError, namespace, "tconfig", err.Error()))
		return reconclie.RateLimit
	}
	for _, tconfig := range tconfigs {
		v := tconfig.(k8sMetaV1.Object)
		name := v.GetName()
		patchContent := []byte("[{\"op\":\"remove\",\"path\":\"/metadata/labels/tars.io~1Deactivate\"},{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Activated\",\"value\":\"false\"},{\"op\":\"replace\",\"path\":\"/activated\",\"value\":false}]")
		_, err = r.clients.CrdClient.CrdV1beta1().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourcePatchError, "tconfig", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
	}
	return reconclie.AllOk
}
