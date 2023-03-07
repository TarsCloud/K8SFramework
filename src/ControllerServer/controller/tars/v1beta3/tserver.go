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
	"strings"
	"tarscontroller/controller"
	"time"
)

type TServerReconciler struct {
	podLister k8sCoreListerV1.PodLister
	tsLister  tarsListerV1beta3.TServerLister
	threads   int
	queue     workqueue.RateLimitingInterface
	synced    []cache.InformerSynced
}

func NewTServerController(threads int) *TServerReconciler {
	podInformer := tarsRuntime.Factories.K8SInformerFactoryWithTarsFilter.Core().V1().Pods()
	tsInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TServers()
	c := &TServerReconciler{
		podLister: podInformer.Lister(),
		tsLister:  tsInformer.Lister(),
		threads:   threads,
		queue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:    []cache.InformerSynced{tsInformer.Informer().HasSynced},
	}
	controller.RegistryInformerEventHandle(tarsMeta.KPodKind, podInformer.Informer(), c)
	controller.RegistryInformerEventHandle(tarsMeta.TServerKind, tsInformer.Informer(), c)
	return c
}

func (r *TServerReconciler) processItem() bool {

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

	res := r.reconcile(key)

	switch res {
	case controller.Done:
		r.queue.Forget(obj)
		return true
	case controller.Retry:
		r.queue.AddRateLimited(obj)
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

func (r *TServerReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
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
		r.queue.Add(key)
	default:
		return
	}
}

func (r *TServerReconciler) Run(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("tserver controller", stopCh, r.synced...) {
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

func (r *TServerReconciler) reconcile(key string) controller.Result {
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
		return controller.Done
	}

	appRequire, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.Equals, []string{tserver.Spec.App})
	serverRequire, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.Equals, []string{tserver.Spec.Server})
	selector := labels.NewSelector().Add(*appRequire).Add(*serverRequire)

	pods, err := r.podLister.Pods(namespace).List(selector)

	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceSelectorError, namespace, "pods", err.Error())
			return controller.Retry
		}
		return controller.Done
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
	tserverCopy.Status = tarsV1beta3.TServerStatus{
		Selector:        selector.String(),
		Replicas:        tserver.Spec.K8S.Replicas,
		ReadyReplicas:   readySize,
		CurrentReplicas: currentSize,
	}
	_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TServers(namespace).UpdateStatus(context.TODO(), tserverCopy, k8sMetaV1.UpdateOptions{})
	if err != nil {
		klog.Errorf(tarsMeta.ResourcePatchError, "tserver", namespace, name, err.Error())
		return controller.Retry
	}
	return controller.Done
}
