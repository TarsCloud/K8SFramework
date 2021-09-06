package v1alpha2

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
	crdV1alpha2 "k8s.tars.io/api/crd/v1alpha2"
	"strings"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	"time"
)

type TEndpointReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTEndpointReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *TEndpointReconciler {
	reconcile := &TEndpointReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
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

	var res reconclie.ReconcileResult
	res = r.reconcile(key)

	switch res {
	case reconclie.AllOk:
		r.workQueue.Forget(obj)
		return true
	case reconclie.RateLimit:
		//r.workQueue.AddRateLimited(obj)
		r.workQueue.AddAfter(obj, time.Millisecond*100)
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

func (r *TEndpointReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1alpha2.TServer:
		tserver := resourceObj.(*crdV1alpha2.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *crdV1alpha2.TEndpoint:
		tendpoint := resourceObj.(*crdV1alpha2.TEndpoint)
		key := fmt.Sprintf("%s/%s", tendpoint.Namespace, tendpoint.Name)
		r.workQueue.Add(key)
	case *k8sCoreV1.Pod:
		pod := resourceObj.(*k8sCoreV1.Pod)
		if pod.Labels != nil {
			app, appExist := pod.Labels[meta.TServerAppLabel]
			server, serverExist := pod.Labels[meta.TServerNameLabel]
			if appExist && serverExist {
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

func (r *TEndpointReconciler) reconcile(key string) reconclie.ReconcileResult {
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
		err = r.clients.CrdClient.CrdV1alpha2().TEndpoints(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "tendpoint", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	if tserver.DeletionTimestamp != nil {
		err = r.clients.CrdClient.CrdV1alpha2().TEndpoints(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "tendpoint", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	tendpoint, err := r.informers.TEndpointInformer.Lister().TEndpoints(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "tendpoint", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		tendpoint = meta.BuildTEndpoint(tserver)
		tendpointInterface := r.clients.CrdClient.CrdV1alpha2().TEndpoints(namespace)
		if _, err = tendpointInterface.Create(context.TODO(), tendpoint, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceCreateError, "tendpoint", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	if !k8sMetaV1.IsControlledBy(tendpoint, tserver) {
		// 此处意味着出现了非由 controller 管理的同名 tendpoint, 需要警告和重试
		msg := fmt.Sprintf(meta.ResourceOutControlError, "tendpoint", namespace, tendpoint.Name, namespace, name)
		meta.Event(namespace, tserver, k8sCoreV1.EventTypeWarning, meta.ResourceOutControlReason, msg)
		return reconclie.RateLimit
	}

	anyChanged := !meta.EqualTServerAndTEndpoint(tserver, tendpoint)

	if anyChanged {
		tendpointCopy := tendpoint.DeepCopy()
		meta.SyncTEndpoint(tserver, tendpointCopy)
		tendpointInterface := r.clients.CrdClient.CrdV1alpha2().TEndpoints(namespace)
		if _, err := tendpointInterface.Update(context.TODO(), tendpointCopy, k8sMetaV1.UpdateOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceUpdateError, "tendpoint", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
	}
	return r.updateStatus(tendpoint)
}

func (r *TEndpointReconciler) buildPodStatus(pod *k8sCoreV1.Pod) *crdV1alpha2.TEndpointPodStatus {
	podStatus := &crdV1alpha2.TEndpointPodStatus{
		UID:               string(pod.UID),
		Name:              pod.Name,
		PodIP:             pod.Status.PodIP,
		HostIP:            pod.Status.HostIP,
		StartTime:         pod.CreationTimestamp,
		ContainerStatuses: pod.Status.ContainerStatuses,
		ID:                pod.Labels[meta.TServerIdLabel],
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
		case meta.TPodReadinessGate:
			tarsReadinessGatesCondition = condition.DeepCopy()
		default:
			podConditions = append(podConditions, condition.DeepCopy())
		}
	}

	if readyConditions != nil {
		if readyConditions.Status == k8sCoreV1.ConditionTrue {
			podStatus.SettingState = "Active"
			podStatus.PresentState = "Active"
			podStatus.PresentMessage = ""
			return podStatus
		}

		if tarsReadinessGatesCondition == nil {
			podStatus.SettingState = "Active"
			podStatus.PresentState = readyConditions.Reason
			podStatus.PresentMessage = readyConditions.Message
			return podStatus
		}
	}

	var presentState string
	podStatus.SettingState = "Active"
	if tarsReadinessGatesCondition != nil {
		v := strings.Split(tarsReadinessGatesCondition.Reason, "/")
		if len(v) < 2 {
			podStatus.SettingState = "Unknown"
			presentState = "Unknown"
		} else {
			podStatus.SettingState = v[0]
			presentState = v[1]
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

	if presentState != "" {
		podStatus.PresentState = presentState
		podStatus.PresentMessage = ""
	}
	return podStatus
}

func (r *TEndpointReconciler) updateStatus(tendpoint *crdV1alpha2.TEndpoint) reconclie.ReconcileResult {
	namespace := tendpoint.Namespace
	appRequirement, _ := labels.NewRequirement(meta.TServerAppLabel, selection.DoubleEquals, []string{tendpoint.Spec.App})
	serverRequirement, _ := labels.NewRequirement(meta.TServerNameLabel, selection.DoubleEquals, []string{tendpoint.Spec.Server})

	pods, err := r.informers.PodInformer.Lister().Pods(namespace).List(labels.NewSelector().Add(*appRequirement).Add(*serverRequirement))
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceSelectorError, namespace, "tendpoint", err.Error()))
		return reconclie.RateLimit
	}

	tendpointPodStatuses := make([]*crdV1alpha2.TEndpointPodStatus, 0, len(pods))
	for _, pod := range pods {
		tendpointPodStatuses = append(tendpointPodStatuses, r.buildPodStatus(pod))
	}

	tendpointCopy := tendpoint.DeepCopy()
	tendpointCopy.Status.PodStatus = tendpointPodStatuses

	_, err = r.clients.CrdClient.CrdV1alpha2().TEndpoints(namespace).UpdateStatus(context.TODO(), tendpointCopy, k8sMetaV1.UpdateOptions{})

	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceUpdateError, "tendpoint", namespace, tendpoint.Name, err.Error()))
		return reconclie.RateLimit
	}

	return reconclie.AllOk
}
