package v1alpha2

import (
	"context"
	"fmt"
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	crdV1alpha2 "k8s.tars.io/api/crd/v1alpha2"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	"time"
)

type DaemonSetReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewDaemonSetReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *DaemonSetReconciler {
	reconcile := &DaemonSetReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *DaemonSetReconciler) processItem() bool {

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
	case reconclie.AddAfter:
		r.workQueue.AddAfter(obj, time.Second*1)
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

func (r *DaemonSetReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1alpha2.TServer:
		tserver := resourceObj.(*crdV1alpha2.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *k8sAppsV1.DaemonSet:
		daemonset := resourceObj.(*k8sAppsV1.DaemonSet)
		if resourceEvent == k8sWatchV1.Deleted || daemonset.DeletionTimestamp != nil {
			key := fmt.Sprintf("%s/%s", daemonset.Namespace, daemonset.Name)
			r.workQueue.Add(key)
		}
	default:
		return
	}
}

func (r *DaemonSetReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *DaemonSetReconciler) reconcile(key string) reconclie.ReconcileResult {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return reconclie.AllOk
	}

	tserver, err := r.informers.TServerInformer.Lister().TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "tserver", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		err = r.clients.K8sClient.AppsV1().DaemonSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "daemonset", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	if tserver.DeletionTimestamp != nil || !tserver.Spec.K8S.DaemonSet {
		err = r.clients.K8sClient.AppsV1().DaemonSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "daemonset", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	daemonSet, err := r.informers.DaemonSetInformer.Lister().DaemonSets(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "daemonset", namespace, name, err.Error()))
			return reconclie.RateLimit
		}

		daemonSet = meta.BuildDaemonSet(tserver)
		daemonSetInterface := r.clients.K8sClient.AppsV1().DaemonSets(namespace)
		if _, err = daemonSetInterface.Create(context.TODO(), daemonSet, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceCreateError, "daemonset", namespace, name, err.Error()))
			return reconclie.RateLimit
		}

		return reconclie.AllOk
	}

	if daemonSet.DeletionTimestamp != nil {
		return reconclie.AddAfter
	}

	if !k8sMetaV1.IsControlledBy(daemonSet, tserver) {
		//此处意味着出现了非由 controller 管理的同名 daemonSet, 需要警告和重试
		msg := fmt.Sprintf(meta.ResourceOutControlError, "daemonset", namespace, daemonSet.Name, namespace, name)
		meta.Event(namespace, tserver, k8sCoreV1.EventTypeWarning, meta.ResourceOutControlReason, msg)
		return reconclie.RateLimit
	}

	anyChanged := !meta.EqualTServerAndDaemonSet(tserver, daemonSet)

	if anyChanged {
		daemonSetCopy := daemonSet.DeepCopy()
		meta.SyncDaemonSet(tserver, daemonSetCopy)
		daemonSetInterface := r.clients.K8sClient.AppsV1().DaemonSets(namespace)
		if _, err := daemonSetInterface.Update(context.TODO(), daemonSetCopy, k8sMetaV1.UpdateOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceUpdateError, "daemonset", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
	}
	return reconclie.AllOk
}
