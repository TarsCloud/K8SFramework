package v1beta3

import (
	"context"
	"fmt"
	"golang.org/x/time/rate"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

type TTreeReconciler struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTTreeReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TTreeReconciler {
	rateLimiter := &workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 50)}
	reconciler := &TTreeReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewRateLimitingQueue(rateLimiter),
	}
	informers.Register(reconciler)
	return reconciler
}

func (r *TTreeReconciler) processItem() bool {

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
	case reconcile.Done:
		r.workQueue.Forget(obj)
		return true
	case reconcile.Retry:
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

func (r *TTreeReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Spec.App)
		r.workQueue.Add(key)
	default:
		return
	}
}

func (r *TTreeReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *TTreeReconciler) reconcile(key string) reconcile.Result {
	namespace, app, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return reconcile.Done
	}

	ttree, err := r.informers.TTreeInformer.Lister().TTrees(namespace).Get(tarsMeta.FixedTTreeResourceName)
	if err != nil {
		msg := fmt.Sprintf(tarsMeta.ResourceGetError, "ttree", namespace, tarsMeta.FixedTTreeResourceName, err.Error())
		utilRuntime.HandleError(fmt.Errorf(msg))
		return reconcile.Retry
	}

	for i := range ttree.Apps {
		if ttree.Apps[i].Name == app {
			return reconcile.Done
		}
	}

	newTressApp := &tarsCrdV1beta3.TTreeApp{
		Name:         app,
		BusinessRef:  "",
		CreatePerson: "",
		CreateTime:   k8sMetaV1.Now(),
		Mark:         "AddByController",
	}
	jsonPatch := tarsMeta.JsonPatch{
		{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/apps/-",
			Value: newTressApp,
		},
	}

	patchContent, _ := json.Marshal(jsonPatch)
	_, err = r.clients.CrdClient.CrdV1beta3().TTrees(namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
	if err != nil {
		return reconcile.Retry
	}

	return reconcile.Done
}
