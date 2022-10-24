package v1beta3

import (
	"context"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	k8sCoreTypeV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sCoreListerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	tarsCrdListerV1beta3 "k8s.tars.io/client-go/listers/crd/v1beta3"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/controller"
	"tarscontroller/util"
	"time"
)

type ServiceReconciler struct {
	clients       *util.Clients
	svcLiter      k8sCoreListerV1.ServiceLister
	tsLister      tarsCrdListerV1beta3.TServerLister
	threads       int
	queue         workqueue.RateLimitingInterface
	synced        []cache.InformerSynced
	eventRecorder record.EventRecorder
}

func NewServiceController(clients *util.Clients, factories *util.InformerFactories, threads int) *ServiceReconciler {
	svcInformer := factories.K8SInformerFactoryWithTarsFilter.Core().V1().Services()
	tsInformer := factories.TarsInformerFactory.Crd().V1beta3().TServers()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8sCoreTypeV1.EventSinkImpl{Interface: clients.K8sClient.CoreV1().Events("")})

	c := &ServiceReconciler{
		clients:       clients,
		svcLiter:      svcInformer.Lister(),
		tsLister:      tsInformer.Lister(),
		threads:       threads,
		queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		synced:        []cache.InformerSynced{svcInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
		eventRecorder: eventBroadcaster.NewRecorder(scheme.Scheme, k8sCoreV1.EventSource{Component: "daemonset-controller"}),
	}
	controller.SetInformerHandlerEvent(tarsMeta.KServiceKind, svcInformer.Informer(), c)
	controller.SetInformerHandlerEvent(tarsMeta.TServerKind, tsInformer.Informer(), c)
	return c
}

func (r *ServiceReconciler) processItem() bool {

	obj, shutdown := r.queue.Get()

	if shutdown {
		return false
	}

	defer r.queue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		utilRuntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		r.queue.Forget(obj)
		return true
	}

	res := r.reconcile(key)

	switch res {
	case controller.Done:
		r.queue.Forget(obj)
		return true
	case controller.Retry:
		r.queue.AddRateLimited(obj)
		return true
	case controller.AddAfter:
		r.queue.AddAfter(obj, time.Second*1)
		return true
	case controller.FatalError:
		r.queue.ShutDown()
		return false
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *ServiceReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.queue.Add(key)
	case *k8sCoreV1.Service:
		service := resourceObj.(*k8sCoreV1.Service)
		if resourceEvent == k8sWatchV1.Deleted {
			key := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
			r.queue.Add(key)
		}
	default:
		return
	}
}

func (r *ServiceReconciler) Run(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("service controller", stopCh, r.synced...) {
		return
	}

	for i := 0; i < r.threads; i++ {
		worker := func() {
			for r.processItem() {
			}
			r.queue.ShutDown()
		}
		go wait.Until(worker, time.Second, stopCh)
	}

	<-stopCh
}

func (r *ServiceReconciler) reconcile(key string) controller.Result {

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return controller.Done
	}

	tserver, err := r.tsLister.TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, name, err.Error()))
			return controller.Retry
		}
		err = r.clients.K8sClient.CoreV1().Services(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "service", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	if tserver.DeletionTimestamp != nil || tserver.Spec.K8S.DaemonSet {
		err = r.clients.K8sClient.CoreV1().Services(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "service", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	service, err := r.svcLiter.Services(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "service", namespace, name, err.Error()))
			return controller.Retry
		}
		service = buildService(tserver)
		serviceInterface := r.clients.K8sClient.CoreV1().Services(namespace)
		if _, err = serviceInterface.Create(context.TODO(), service, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceCreateError, "service", namespace, name, err.Error()))
			return controller.Retry
		}

		return controller.Done
	}

	if service.DeletionTimestamp != nil {
		return controller.AddAfter
	}

	if !k8sMetaV1.IsControlledBy(service, tserver) {
		// 此处意味着出现了非由 controller 管理的同名 service, 需要警告和重试
		msg := fmt.Sprintf(tarsMeta.ResourceOutControlError, "service", namespace, service.Name, namespace, name)
		r.eventRecorder.Event(tserver, k8sCoreV1.EventTypeWarning, tarsMeta.ResourceOutControlReason, msg)
		return controller.Retry
	}

	anyChanged := !EqualTServerAndService(tserver, service)

	if anyChanged {
		serviceCopy := service.DeepCopy()
		syncService(tserver, serviceCopy)
		serviceInterface := r.clients.K8sClient.CoreV1().Services(namespace)
		if _, err := serviceInterface.Update(context.TODO(), serviceCopy, k8sMetaV1.UpdateOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "service", namespace, name, err.Error()))
			return controller.Retry
		}
	}

	return controller.Done
}
