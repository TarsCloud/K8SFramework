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
	"k8s.io/klog/v2"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsListerV1beta3 "k8s.tars.io/client-go/listers/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"strings"
	"tarscontroller/controller"
	"time"
)

type TEndpointReconciler struct {
	podLister k8sCoreListerV1.PodLister
	teLister  tarsListerV1beta3.TEndpointLister
	tsLister  tarsListerV1beta3.TServerLister
	threads   int
	queue     workqueue.RateLimitingInterface
	synced    []cache.InformerSynced
}

func NewTEndpointController(threads int) *TEndpointReconciler {
	podInformer := tarsRuntime.Factories.K8SInformerFactoryWithTarsFilter.Core().V1().Pods()
	teInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TEndpoints()
	tsInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TServers()
	c := &TEndpointReconciler{
		podLister: podInformer.Lister(),
		teLister:  teInformer.Lister(),
		tsLister:  tsInformer.Lister(),
		threads:   threads,
		queue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:    []cache.InformerSynced{podInformer.Informer().HasSynced, teInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
	}
	controller.RegistryInformerEventHandle(tarsMeta.KPodKind, podInformer.Informer(), c)
	controller.RegistryInformerEventHandle(tarsMeta.TEndpointKind, teInformer.Informer(), c)
	controller.RegistryInformerEventHandle(tarsMeta.TServerKind, tsInformer.Informer(), c)
	return c
}

func (r *TEndpointReconciler) processItem() bool {

	obj, shutdown := r.queue.Get()

	if shutdown {
		return false
	}

	defer r.queue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		klog.Errorf("expected string in workqueue but got %#v", obj)
		r.queue.Forget(obj)
		return true
	}

	var res controller.Result
	res = r.reconcile(key)

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
		klog.Errorf("should not reach place")
		return false
	}
}

func (r *TEndpointReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsV1beta3.TServer:
		tserver := resourceObj.(*tarsV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.queue.Add(key)
	case *tarsV1beta3.TEndpoint:
		tendpoint := resourceObj.(*tarsV1beta3.TEndpoint)
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

func (r *TEndpointReconciler) Run(stopCh chan struct{}) {
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

func (r *TEndpointReconciler) reconcile(key string) controller.Result {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)

	if err != nil {
		klog.Errorf("invalid key: %s", key)
		return controller.Done
	}

	tserver, err := r.tsLister.TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, name, err.Error())
			return controller.Retry
		}
		err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TEndpoints(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceDeleteError, "tendpoint", namespace, name, err.Error())
			return controller.Retry
		}
		return controller.Done
	}

	if tserver.DeletionTimestamp != nil {
		err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TEndpoints(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceDeleteError, "tendpoint", namespace, name, err.Error())
			return controller.Retry
		}
		return controller.Done
	}

	tendpoint, err := r.teLister.TEndpoints(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceGetError, "tendpoint", namespace, name, err.Error())
			return controller.Retry
		}
		tendpoint = tarsRuntime.TarsTranslator.BuildTEndpoint(tserver)
		tendpointInterface := tarsRuntime.Clients.CrdClient.TarsV1beta3().TEndpoints(namespace)
		if _, err = tendpointInterface.Create(context.TODO(), tendpoint, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			klog.Errorf(tarsMeta.ResourceCreateError, "tendpoint", namespace, name, err.Error())
			return controller.Retry
		}
		return controller.Done
	}

	if !k8sMetaV1.IsControlledBy(tendpoint, tserver) {
		tendpointInterface := tarsRuntime.Clients.CrdClient.TarsV1beta3().TEndpoints(namespace)
		if err = tendpointInterface.Delete(context.TODO(), tendpoint.Name, k8sMetaV1.DeleteOptions{}); err != nil {
			klog.Errorf(tarsMeta.ResourceUpdateError, "tendpoint", namespace, name, err.Error())
		}
		return controller.Retry
	}

	update, target := tarsRuntime.TarsTranslator.DryRunSyncTEndpoint(tserver, tendpoint)
	if update {
		tendpointInterface := tarsRuntime.Clients.CrdClient.TarsV1beta3().TEndpoints(namespace)
		if _, err = tendpointInterface.Update(context.TODO(), target, k8sMetaV1.UpdateOptions{}); err != nil {
			klog.Errorf(tarsMeta.ResourceUpdateError, "tendpoint", namespace, name, err.Error())
			return controller.Retry
		}
	}
	return r.updateStatus(tendpoint)
}

func (r *TEndpointReconciler) buildPodStatus(pod *k8sCoreV1.Pod) *tarsV1beta3.TEndpointPodStatus {
	podStatus := &tarsV1beta3.TEndpointPodStatus{
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
			podStatus.SettingState = "Active"
			podStatus.PresentState = "Active"
			if tarsReadinessGatesCondition != nil {
				podStatus.PresentMessage = readyConditions.Message
				_, _, pid := tarsTool.SplitReadinessConditionReason(tarsReadinessGatesCondition.Reason)
				podStatus.PID = pid
			}
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
		podStatus.SettingState, podStatus.PresentState, podStatus.PID = tarsTool.SplitReadinessConditionReason(tarsReadinessGatesCondition.Reason)
	}
	return podStatus
}

func (r *TEndpointReconciler) updateStatus(tendpoint *tarsV1beta3.TEndpoint) controller.Result {
	namespace := tendpoint.Namespace
	appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tendpoint.Spec.App})
	serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{tendpoint.Spec.Server})

	pods, err := r.podLister.Pods(namespace).List(labels.NewSelector().Add(*appRequirement).Add(*serverRequirement))
	if err != nil {
		klog.Errorf(tarsMeta.ResourceSelectorError, namespace, "tendpoint", err.Error())
		return controller.Retry
	}

	tendpointPodStatuses := make([]*tarsV1beta3.TEndpointPodStatus, 0, len(pods))
	for _, pod := range pods {
		tendpointPodStatuses = append(tendpointPodStatuses, r.buildPodStatus(pod))
	}

	tendpointCopy := tendpoint.DeepCopy()
	tendpointCopy.Status.PodStatus = tendpointPodStatuses

	_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TEndpoints(namespace).UpdateStatus(context.TODO(), tendpointCopy, k8sMetaV1.UpdateOptions{})
	if err != nil {
		klog.Errorf(tarsMeta.ResourceUpdateError, "tendpoint", namespace, tendpoint.Name, err.Error())
		return controller.Retry
	}

	return controller.Done
}
