package v1beta3

import (
	"context"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

type TServerReconciler struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTServerReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TServerReconciler {
	reconciler := &TServerReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	informers.Register(reconciler)
	return reconciler
}

func (r *TServerReconciler) processItem() bool {

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

func (r *TServerReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *k8sCoreV1.Pod:
		pod := resourceObj.(*k8sCoreV1.Pod)
		if pod.Labels == nil {
			return
		}
		var app, server string
		var ok bool
		if app, ok = pod.Labels[tarsMeta.TServerAppLabel]; !ok && app != "" {
			return
		}
		if server, ok = pod.Labels[tarsMeta.TServerNameLabel]; !ok && server != "" {
			return
		}
		key := fmt.Sprintf("%s/%s-%s", pod.Namespace, strings.ToLower(app), strings.ToLower(server))
		r.workQueue.Add(key)
	default:
		return
	}
}

func (r *TServerReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *TServerReconciler) reconcile(key string) reconcile.Result {
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
		return reconcile.Done
	}

	appRequire, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.Equals, []string{tserver.Spec.App})
	serverRequire, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.Equals, []string{tserver.Spec.Server})
	selector := labels.NewSelector().Add(*appRequire).Add(*serverRequire)

	pods, err := r.informers.PodInformer.Lister().Pods(namespace).List(selector)

	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "pods", err.Error()))
			return reconcile.Retry
		}
		return reconcile.Done
	}

	var readySize int32 = 0
	var currentSize int32 = 0
	for _, pod := range pods {
		currentSize += 1
		if pod.Status.Conditions != nil {
			for _, condition := range pod.Status.Conditions {
				if condition.Type == k8sCoreV1.PodReady && condition.Status == k8sCoreV1.ConditionTrue {
					readySize += 1
					break
				}
			}
		}
	}

	tserverCopy := tserver.DeepCopy()
	tserverCopy.Status = tarsCrdV1beta3.TServerStatus{
		Selector:        selector.String(),
		Replicas:        tserver.Spec.K8S.Replicas,
		ReadyReplicas:   readySize,
		CurrentReplicas: currentSize,
	}
	_, err = r.clients.CrdClient.CrdV1beta3().TServers(namespace).UpdateStatus(context.TODO(), tserverCopy, k8sMetaV1.UpdateOptions{})
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourcePatchError, "tserver", namespace, name, err.Error()))
		return reconcile.Retry
	}
	return reconcile.Done
}
