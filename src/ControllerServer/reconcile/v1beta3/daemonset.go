package v1beta3

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
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

type DaemonSetReconciler struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewDaemonSetReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *DaemonSetReconciler {
	reconciler := &DaemonSetReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	informers.Register(reconciler)
	return reconciler
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
	case reconcile.Done:
		r.workQueue.Forget(obj)
		return true
	case reconcile.Retry:
		r.workQueue.AddRateLimited(obj)
		return true
	case reconcile.AddAfter:
		r.workQueue.AddAfter(obj, time.Second*1)
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

func (r *DaemonSetReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
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

func (r *DaemonSetReconciler) reconcile(key string) reconcile.Result {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return reconcile.Done
	}

	tserver, err := r.informers.TServerInformer.Lister().TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, name, err.Error()))
			return reconcile.Retry
		}
		err = r.clients.K8sClient.AppsV1().DaemonSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "daemonset", namespace, name, err.Error()))
			return reconcile.Retry
		}
		return reconcile.Done
	}

	if tserver.DeletionTimestamp != nil || !tserver.Spec.K8S.DaemonSet {
		err = r.clients.K8sClient.AppsV1().DaemonSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "daemonset", namespace, name, err.Error()))
			return reconcile.Retry
		}
		return reconcile.Done
	}

	daemonSet, err := r.informers.DaemonSetInformer.Lister().DaemonSets(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "daemonset", namespace, name, err.Error()))
			return reconcile.Retry
		}

		daemonSet = buildDaemonset(tserver)
		daemonSetInterface := r.clients.K8sClient.AppsV1().DaemonSets(namespace)
		if _, err = daemonSetInterface.Create(context.TODO(), daemonSet, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceCreateError, "daemonset", namespace, name, err.Error()))
			return reconcile.Retry
		}

		return reconcile.Done
	}

	if daemonSet.DeletionTimestamp != nil {
		return reconcile.AddAfter
	}

	if !k8sMetaV1.IsControlledBy(daemonSet, tserver) {
		//此处意味着出现了非由 controller 管理的同名 daemonSet, 需要警告和重试
		msg := fmt.Sprintf(tarsMeta.ResourceOutControlError, "daemonset", namespace, daemonSet.Name, namespace, name)
		controller.Event(tserver, k8sCoreV1.EventTypeWarning, tarsMeta.ResourceOutControlReason, msg)
		return reconcile.Retry
	}

	anyChanged := !EqualTServerAndDaemonSet(tserver, daemonSet)

	if anyChanged {
		daemonSetCopy := daemonSet.DeepCopy()
		syncDaemonSet(tserver, daemonSetCopy)
		daemonSetInterface := r.clients.K8sClient.AppsV1().DaemonSets(namespace)
		if _, err := daemonSetInterface.Update(context.TODO(), daemonSetCopy, k8sMetaV1.UpdateOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "daemonset", namespace, name, err.Error()))
			return reconcile.Retry
		}
	}
	return reconcile.Done
}
