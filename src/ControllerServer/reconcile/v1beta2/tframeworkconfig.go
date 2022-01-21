package v1beta2

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

type TFrameworkConfigReconciler struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTFrameworkConfigReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TFrameworkConfigReconciler {
	reconciler := &TFrameworkConfigReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconciler)
	return reconciler
}

func (r *TFrameworkConfigReconciler) processItem() bool {

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

func (r *TFrameworkConfigReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1beta2.TFrameworkConfig:
		tfc := resourceObj.(*crdV1beta2.TFrameworkConfig)
		key := fmt.Sprintf("%s/%s", tfc.Namespace, tfc.Name)
		r.workQueue.Add(key)
	default:
		return
	}
}

func (r *TFrameworkConfigReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *TFrameworkConfigReconciler) reconcile(key string) reconcile.Result {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return reconcile.AllOk
	}

	if name != crdMeta.FixedTFrameworkConfigResourceName {
		return reconcile.AllOk
	}

	tframeworkconfig, err := r.informers.TFrameworkConfigInformer.Lister().TFrameworkConfigs(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceGetError, "tfameworkconfig", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	if tframeworkconfig.DeletionTimestamp != nil {
		return reconcile.AllOk
	}

	controller.SetTFrameworkConfig(namespace, tframeworkconfig)
	return reconcile.AllOk
}
