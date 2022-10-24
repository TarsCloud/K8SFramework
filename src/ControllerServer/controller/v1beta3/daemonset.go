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
	"k8s.io/client-go/kubernetes/scheme"
	k8sCoreTypeV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sAppsListerV1 "k8s.io/client-go/listers/apps/v1"
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

type DaemonSetReconciler struct {
	clients       *util.Clients
	dsLister      k8sAppsListerV1.DaemonSetLister
	tsLister      tarsCrdListerV1beta3.TServerLister
	threads       int
	queue         workqueue.RateLimitingInterface
	synced        []cache.InformerSynced
	eventRecorder record.EventRecorder
}

func NewDaemonSetController(clients *util.Clients, factories *util.InformerFactories, threads int) *DaemonSetReconciler {
	dsInformer := factories.K8SInformerFactoryWithTarsFilter.Apps().V1().DaemonSets()
	tsInformer := factories.TarsInformerFactory.Crd().V1beta3().TServers()

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8sCoreTypeV1.EventSinkImpl{Interface: clients.K8sClient.CoreV1().Events("")})

	c := &DaemonSetReconciler{
		clients:       clients,
		dsLister:      dsInformer.Lister(),
		tsLister:      tsInformer.Lister(),
		threads:       threads,
		queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:        []cache.InformerSynced{dsInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
		eventRecorder: eventBroadcaster.NewRecorder(scheme.Scheme, k8sCoreV1.EventSource{Component: "daemonset-controller"}),
	}

	controller.SetInformerHandlerEvent(tarsMeta.KDaemonSetKind, dsInformer.Informer(), c)
	controller.SetInformerHandlerEvent(tarsMeta.TServerKind, tsInformer.Informer(), c)

	return c
}

func (r *DaemonSetReconciler) processItem() bool {

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

func (r *DaemonSetReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.queue.Add(key)
	case *k8sAppsV1.DaemonSet:
		daemonset := resourceObj.(*k8sAppsV1.DaemonSet)
		if resourceEvent == k8sWatchV1.Deleted {
			key := fmt.Sprintf("%s/%s", daemonset.Namespace, daemonset.Name)
			r.queue.Add(key)
		}
	default:
		return
	}
}

func (r *DaemonSetReconciler) Run(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("daemonset controller", stopCh, r.synced...) {
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

func (r *DaemonSetReconciler) reconcile(key string) controller.Result {
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
		err = r.clients.K8sClient.AppsV1().DaemonSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "daemonset", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	if tserver.DeletionTimestamp != nil || !tserver.Spec.K8S.DaemonSet {
		err = r.clients.K8sClient.AppsV1().DaemonSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "daemonset", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	daemonSet, err := r.dsLister.DaemonSets(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "daemonset", namespace, name, err.Error()))
			return controller.Retry
		}

		daemonSet = buildDaemonset(tserver)
		daemonSetInterface := r.clients.K8sClient.AppsV1().DaemonSets(namespace)
		if _, err = daemonSetInterface.Create(context.TODO(), daemonSet, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceCreateError, "daemonset", namespace, name, err.Error()))
			return controller.Retry
		}

		return controller.Done
	}

	if daemonSet.DeletionTimestamp != nil {
		return controller.AddAfter
	}

	if !k8sMetaV1.IsControlledBy(daemonSet, tserver) {
		//此处意味着出现了非由 controller 管理的同名 daemonSet, 需要警告和重试
		msg := fmt.Sprintf(tarsMeta.ResourceOutControlError, "daemonset", namespace, daemonSet.Name, namespace, name)
		r.eventRecorder.Event(tserver, k8sCoreV1.EventTypeWarning, tarsMeta.ResourceOutControlReason, msg)
		return controller.Retry
	}

	anyChanged := !EqualTServerAndDaemonSet(tserver, daemonSet)

	if anyChanged {
		daemonSetCopy := daemonSet.DeepCopy()
		syncDaemonSet(tserver, daemonSetCopy)
		daemonSetInterface := r.clients.K8sClient.AppsV1().DaemonSets(namespace)
		if _, err := daemonSetInterface.Update(context.TODO(), daemonSetCopy, k8sMetaV1.UpdateOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "daemonset", namespace, name, err.Error()))
			return controller.Retry
		}
	}
	return controller.Done
}
