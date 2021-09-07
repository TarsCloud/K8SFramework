package v1beta1

import (
	"context"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	crdV1beta1 "k8s.tars.io/api/crd/v1beta1"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	"time"
)

type ServiceReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewServiceReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *ServiceReconciler {
	reconcile := &ServiceReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *ServiceReconciler) processItem() bool {

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

func (r *ServiceReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1beta1.TServer:
		tserver := resourceObj.(*crdV1beta1.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *k8sCoreV1.Service:
		service := resourceObj.(*k8sCoreV1.Service)
		if resourceEvent == k8sWatchV1.Deleted || service.DeletionTimestamp != nil {
			key := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
			r.workQueue.Add(key)
		}
	default:
		return
	}
}

func (r *ServiceReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *ServiceReconciler) reconcile(key string) reconclie.ReconcileResult {

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
		err = r.clients.K8sClient.CoreV1().Services(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "service", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	if tserver.DeletionTimestamp != nil || tserver.Spec.K8S.DaemonSet {
		err = r.clients.K8sClient.CoreV1().Services(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "service", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	service, err := r.informers.ServiceInformer.Lister().Services(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "service", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		service = meta.BuildService(tserver)
		serviceInterface := r.clients.K8sClient.CoreV1().Services(namespace)
		if _, err = serviceInterface.Create(context.TODO(), service, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceCreateError, "service", namespace, name, err.Error()))
			return reconclie.RateLimit
		}

		return reconclie.AllOk
	}

	if service.DeletionTimestamp != nil {
		return reconclie.AddAfter
	}

	if !k8sMetaV1.IsControlledBy(service, tserver) {
		// 此处意味着出现了非由 controller 管理的同名 service, 需要警告和重试
		msg := fmt.Sprintf(meta.ResourceOutControlError, "service", namespace, service.Name, namespace, name)
		meta.Event(namespace, tserver, k8sCoreV1.EventTypeWarning, meta.ResourceOutControlReason, msg)
		return reconclie.RateLimit
	}

	anyChanged := !meta.EqualTServerAndService(tserver, service)

	if anyChanged {
		serviceCopy := service.DeepCopy()
		meta.SyncService(tserver, serviceCopy)
		serviceInterface := r.clients.K8sClient.CoreV1().Services(namespace)
		if _, err := serviceInterface.Update(context.TODO(), serviceCopy, k8sMetaV1.UpdateOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceUpdateError, "service", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
	}

	return reconclie.AllOk
}
