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
	k8sCoreListerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	tarsCrdListerV1beta3 "k8s.tars.io/client-go/listers/crd/v1beta3"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/util"
	"time"
)

type TEndpointReconciler struct {
	clients   *util.Clients
	podLister k8sCoreListerV1.PodLister
	teLister  tarsCrdListerV1beta3.TEndpointLister
	tsLister  tarsCrdListerV1beta3.TServerLister
	threads   int
	queue     workqueue.RateLimitingInterface
	synced    []cache.InformerSynced
}

func NewTEndpointController(clients *util.Clients, factories *util.InformerFactories, threads int) *TEndpointReconciler {
	podInformer := factories.K8SInformerFactoryWithTarsFilter.Core().V1().Pods()
	teInformer := factories.TarsInformerFactory.Crd().V1beta3().TEndpoints()
	tsInformer := factories.TarsInformerFactory.Crd().V1beta3().TServers()
	c := &TEndpointReconciler{
		clients:   clients,
		podLister: podInformer.Lister(),
		teLister:  teInformer.Lister(),
		tsLister:  tsInformer.Lister(),
		threads:   threads,
		queue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:    []cache.InformerSynced{podInformer.Informer().HasSynced, teInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
	}
	controller.SetInformerHandlerEvent(tarsMeta.KPodKind, podInformer.Informer(), c)
	controller.SetInformerHandlerEvent(tarsMeta.TEndpointKind, teInformer.Informer(), c)
	controller.SetInformerHandlerEvent(tarsMeta.TServerKind, tsInformer.Informer(), c)
	return c
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

	var res controller.Result
	res = r.sync(key)

	switch res {
	case controller.Done:
		r.queue.Forget(obj)
		return true
	case controller.Retry:
		//r.queue.AddRateLimited(obj)
		r.queue.AddAfter(obj, time.Millisecond*100)
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

func (r *TEndpointReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.queue.Add(key)
	case *tarsCrdV1beta3.TEndpoint:
		tendpoint := resourceObj.(*tarsCrdV1beta3.TEndpoint)
		key := fmt.Sprintf("%s/%s", tendpoint.Namespace, tendpoint.Name)
		r.queue.Add(key)
	case *k8sCoreV1.Pod:
		pod := resourceObj.(*k8sCoreV1.Pod)
		if pod.Labels != nil {
			app := pod.Labels[tarsMeta.TServerAppLabel]
			server := pod.Labels[tarsMeta.TServerNameLabel]
			if app != "" && server != "" {
				key := fmt.Sprintf("%s/%s-%s", pod.Namespace, strings.ToLower(app), strings.ToLower(server))
				r.queue.Add(key)
				return
			}
		}
	default:
		return
	}
}

func (r *TEndpointReconciler) StartController(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("tendpoint controller", stopCh, r.synced...) {
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

func (r *TEndpointReconciler) sync(key string) controller.Result {
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
		err = r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "tendpoint", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	if tserver.DeletionTimestamp != nil {
		err = r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "tendpoint", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	tendpoint, err := r.teLister.TEndpoints(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "tendpoint", namespace, name, err.Error()))
			return controller.Retry
		}
		tendpoint = buildTEndpoint(tserver)
		tendpointInterface := r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace)
		if _, err = tendpointInterface.Create(context.TODO(), tendpoint, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceCreateError, "tendpoint", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	if !k8sMetaV1.IsControlledBy(tendpoint, tserver) {
		tendpointInterface := r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace)
		if err = tendpointInterface.Delete(context.TODO(), tendpoint.Name, k8sMetaV1.DeleteOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "tendpoint", namespace, name, err.Error()))
		}
		return controller.Retry
	}

	anyChanged := !EqualTServerAndTEndpoint(tserver, tendpoint)
	if anyChanged {
		tendpointCopy := tendpoint.DeepCopy()
		syncTEndpoint(tserver, tendpointCopy)
		tendpointInterface := r.clients.CrdClient.CrdV1beta3().TEndpoints(namespace)
		if _, err = tendpointInterface.Update(context.TODO(), tendpointCopy, k8sMetaV1.UpdateOptions{}); err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "tendpoint", namespace, name, err.Error()))
			return controller.Retry
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

func (r *TEndpointReconciler) updateStatus(tendpoint *tarsCrdV1beta3.TEndpoint) controller.Result {
	namespace := tendpoint.Namespace
	appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tendpoint.Spec.App})
	serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{tendpoint.Spec.Server})

	pods, err := r.podLister.Pods(namespace).List(labels.NewSelector().Add(*appRequirement).Add(*serverRequirement))
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "tendpoint", err.Error()))
		return controller.Retry
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
		return controller.Retry
	}

	return controller.Done
}
