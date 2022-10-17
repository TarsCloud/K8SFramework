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

type TEndpointReconciler struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTEndpointReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TEndpointReconciler {
	reconciler := &TEndpointReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconciler)
	return reconciler
}

func splitTARSConditionReason(reason string) (setting, present, pid string) {
	v := strings.Split(reason, "/")
	switch len(v) {
	case 1:
		return v[0], "", ""
	case 2:
		return v[0], v[1], ""
	case 3:
		return v[0], v[1], v[2]
	default:
		return "Unknown", "Unknown", ""
	}
}

func (r *TEndpointReconciler) processItem() bool {

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

	var res reconcile.Result
	res = r.reconcile(key)

	switch res {
	case reconcile.AllOk:
		r.workQueue.Forget(obj)
		return true
	case reconcile.RateLimit:
		//r.workQueue.AddRateLimited(obj)
		r.workQueue.AddAfter(obj, time.Millisecond*100)
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

func (r *TEndpointReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *tarsCrdV1beta3.TEndpoint:
		tendpoint := resourceObj.(*tarsCrdV1beta3.TEndpoint)
		key := fmt.Sprintf("%s/%s", tendpoint.Namespace, tendpoint.Name)
		r.workQueue.Add(key)
	case *k8sCoreV1.Pod:
		pod := resourceObj.(*k8sCoreV1.Pod)
		if pod.Labels != nil {
			app := pod.Labels[tarsMeta.TServerAppLabel]
			server := pod.Labels[tarsMeta.TServerNameLabel]
			if app != "" && server != "" {
				key := fmt.Sprintf("%s/%s-%s", pod.Namespace, strings.ToLower(app), strings.ToLower(server))
				r.workQueue.Add(key)
				return
			}
		}
	default:
		return
	}
}

func (r *TEndpointReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *TEndpointReconciler) reconcile(key string) reconcile.Result {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)

	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return reconcile.AllOk
	}

	tserver, err := r.informers.TServerInformer.Lister().TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		err = r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "tendpoint", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	if tserver.DeletionTimestamp != nil {
		err = r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "tendpoint", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	tendpoint, err := r.informers.TEndpointInformer.Lister().TEndpoints(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "tendpoint", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		tendpoint = buildTEndpoint(tserver)
		tendpointInterface := r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace)
		if _, err = tendpointInterface.Create(context.TODO(), tendpoint, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceCreateError, "tendpoint", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	if !k8sMetaV1.IsControlledBy(tendpoint, tserver) {
		tendpointInterface := r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace)
		if err = tendpointInterface.Delete(context.TODO(), tendpoint.Name, k8sMetaV1.DeleteOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "tendpoint", namespace, name, err.Error()))
		}
		return reconcile.RateLimit
	}

	anyChanged := !EqualTServerAndTEndpoint(tserver, tendpoint)

	if anyChanged {
		tendpointCopy := tendpoint.DeepCopy()
		syncTEndpoint(tserver, tendpointCopy)
		tendpointInterface := r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace)
		if _, err = tendpointInterface.Update(context.TODO(), tendpointCopy, k8sMetaV1.UpdateOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "tendpoint", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
	}
	return r.updateStatus(tendpoint)
}

func (r *TEndpointReconciler) buildPodStatus(pod *k8sCoreV1.Pod) *tarsCrdV1beta3.TEndpointPodStatus {
	podStatus := &tarsCrdV1beta3.TEndpointPodStatus{
		UID:               string(pod.UID),
		PID:               "",
		Name:              pod.Name,
		PodIP:             pod.Status.PodIP,
		HostIP:            pod.Status.HostIP,
		StartTime:         pod.CreationTimestamp,
		ContainerStatuses: pod.Status.ContainerStatuses,
		SettingState:      "Active",
		PresentState:      "",
		PresentMessage:    "",
		ID:                pod.Labels[tarsMeta.TServerIdLabel],
	}

	if pod.DeletionTimestamp != nil {
		podStatus.SettingState = "Active"
		podStatus.PresentState = "Terminating"
		podStatus.PresentMessage = fmt.Sprintf("pod/%s is terminating", pod.Name)
		return podStatus
	}

	podConditions := make([]*k8sCoreV1.PodCondition, 3, 3)
	var readyConditions *k8sCoreV1.PodCondition
	var tarsReadinessGatesCondition *k8sCoreV1.PodCondition

	for _, condition := range pod.Status.Conditions {
		switch condition.Type {
		case k8sCoreV1.PodScheduled:
			podConditions[0] = condition.DeepCopy()
		case k8sCoreV1.PodInitialized:
			podConditions[1] = condition.DeepCopy()
		case k8sCoreV1.ContainersReady:
			podConditions[2] = condition.DeepCopy()
		case k8sCoreV1.PodReady:
			readyConditions = condition.DeepCopy()
		case tarsMeta.TPodReadinessGate:
			tarsReadinessGatesCondition = condition.DeepCopy()
		default:
			podConditions = append(podConditions, condition.DeepCopy())
		}
	}

	if readyConditions != nil {
		if readyConditions.Status == k8sCoreV1.ConditionTrue {

			if tarsReadinessGatesCondition != nil {
				_, _, pid := splitTARSConditionReason(tarsReadinessGatesCondition.Reason)
				podStatus.SettingState = "Active"
				podStatus.PresentState = "Active"
				podStatus.PresentMessage = readyConditions.Message
				podStatus.PID = pid
				return podStatus
			}

			podStatus.SettingState = "Active"
			podStatus.PresentState = "Active"
			podStatus.PresentMessage = ""
			return podStatus
		}
	}

	for _, condition := range podConditions {
		if condition != nil {
			if condition.Status != k8sCoreV1.ConditionTrue {
				podStatus.PresentState = condition.Reason
				podStatus.PresentMessage = condition.Message
				return podStatus
			}
			podStatus.PresentState = string(condition.Type)
			podStatus.PresentMessage = condition.Message
		}
	}

	if tarsReadinessGatesCondition != nil {
		podStatus.SettingState, podStatus.PresentState, podStatus.PID = splitTARSConditionReason(tarsReadinessGatesCondition.Reason)
	}
	return podStatus
}

func (r *TEndpointReconciler) updateStatus(tendpoint *tarsCrdV1beta3.TEndpoint) reconcile.Result {
	namespace := tendpoint.Namespace
	appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tendpoint.Spec.App})
	serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{tendpoint.Spec.Server})

	pods, err := r.informers.PodInformer.Lister().Pods(namespace).List(labels.NewSelector().Add(*appRequirement).Add(*serverRequirement))
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "tendpoint", err.Error()))
		return reconcile.RateLimit
	}

	tendpointPodStatuses := make([]*tarsCrdV1beta3.TEndpointPodStatus, 0, len(pods))
	for _, pod := range pods {
		tendpointPodStatuses = append(tendpointPodStatuses, r.buildPodStatus(pod))
	}

	tendpointCopy := tendpoint.DeepCopy()
	tendpointCopy.Status.PodStatus = tendpointPodStatuses

	_, err = r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace).UpdateStatus(context.TODO(), tendpointCopy, k8sMetaV1.UpdateOptions{})

	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "tendpoint", namespace, tendpoint.Name, err.Error()))
		return reconcile.RateLimit
	}

	return reconcile.AllOk
}
