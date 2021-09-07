package v1beta1

import (
	"context"
	"fmt"
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

type StatefulSetReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func diffVolumeClaimTemplate(currents, targets []k8sCoreV1.PersistentVolumeClaim) (bool, []string) {
	if currents == nil && targets != nil {
		return false, nil
	}

	targetVCS := make(map[string]*k8sCoreV1.PersistentVolumeClaim, len(targets))
	for i := range targets {
		targetVCS[targets[i].Name] = &targets[i]
	}

	var equal = true
	var shouldDelete []string

	for i := range currents {
		c := &currents[i]
		t, ok := targetVCS[c.Name]
		if !ok {
			equal = false
			shouldDelete = append(shouldDelete, c.Name)
			continue
		}

		if equal == true {
			if !equality.Semantic.DeepEqual(c.ObjectMeta, t.ObjectMeta) {
				equal = false
				continue
			}

			if !equality.Semantic.DeepEqual(c.Spec, t.Spec) {
				equal = false
				continue
			}
		}
	}
	return equal, shouldDelete
}

func NewStatefulSetReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *StatefulSetReconciler {
	reconcile := &StatefulSetReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *StatefulSetReconciler) processItem() bool {

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
	case reconclie.FatalError:
		r.workQueue.ShutDown()
		return false
	case reconclie.AddAfter:
		r.workQueue.AddAfter(obj, time.Second*3)
		return true
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *StatefulSetReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1beta1.TServer:
		tserver := resourceObj.(*crdV1beta1.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *k8sAppsV1.StatefulSet:
		statefulset := resourceObj.(*k8sAppsV1.StatefulSet)
		if statefulset.DeletionTimestamp != nil || resourceEvent == k8sWatchV1.Deleted {
			key := fmt.Sprintf("%s/%s", statefulset.Namespace, statefulset.Name)
			r.workQueue.Add(key)
		}
	}
}

func (r *StatefulSetReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *StatefulSetReconciler) syncStatefulset(tserver *crdV1beta1.TServer, statefulSet *k8sAppsV1.StatefulSet, namespace, name string) reconclie.ReconcileResult {
	statefulSetCopy := statefulSet.DeepCopy()
	meta.SyncStatefulSet(tserver, statefulSetCopy)
	statefulSetInterface := r.clients.K8sClient.AppsV1().StatefulSets(namespace)
	if _, err := statefulSetInterface.Update(context.TODO(), statefulSetCopy, k8sMetaV1.UpdateOptions{}); err != nil {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceUpdateError, "statefulset", namespace, name, err.Error()))
		return reconclie.RateLimit
	}
	return reconclie.AllOk
}

func (r *StatefulSetReconciler) recreateStatefulset(tserver *crdV1beta1.TServer, shouldDeletePVCS []string) reconclie.ReconcileResult {
	namespace := tserver.Namespace
	name := tserver.Name
	err := r.clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "statefulset", namespace, name, err.Error()))
		return reconclie.RateLimit
	}

	if shouldDeletePVCS != nil {
		appRequirement, _ := labels.NewRequirement(meta.TServerAppLabel, "==", []string{tserver.Spec.App})
		serverRequirement, _ := labels.NewRequirement(meta.TServerNameLabel, "==", []string{tserver.Spec.Server})
		labelSelector := labels.NewSelector().Add(*appRequirement, *serverRequirement)

		localVolumeRequirement, _ := labels.NewRequirement(meta.TLocalVolumeLabel, "==", shouldDeletePVCS)
		labelSelector.Add(*localVolumeRequirement)

		err = r.clients.K8sClient.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
			LabelSelector: labelSelector.String(),
		})

		if err != nil {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteCollectionError, "persistentvolumeclaims", labelSelector.String(), err.Error()))
		}
	}
	return reconclie.AddAfter
}

func (r *StatefulSetReconciler) reconcile(key string) reconclie.ReconcileResult {
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
		err = r.clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "statefulset", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	if tserver.DeletionTimestamp != nil || tserver.Spec.K8S.DaemonSet {
		err = r.clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "statefulset", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	statefulSet, err := r.informers.StatefulSetInformer.Lister().StatefulSets(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "statefulset", namespace, name, err.Error()))
			return reconclie.RateLimit
		}

		if !tserver.Spec.K8S.DaemonSet {
			statefulSet = meta.BuildStatefulSet(tserver)
			statefulSetInterface := r.clients.K8sClient.AppsV1().StatefulSets(namespace)
			if _, err = statefulSetInterface.Create(context.TODO(), statefulSet, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				utilRuntime.HandleError(fmt.Errorf(meta.ResourceCreateError, "statefulset", namespace, name, err.Error()))
				return reconclie.RateLimit
			}
		}
		return reconclie.AllOk
	}

	if statefulSet.DeletionTimestamp != nil {
		return reconclie.AddAfter
	}

	if !k8sMetaV1.IsControlledBy(statefulSet, tserver) {
		// 此处意味着出现了非由 controller 管理的同名 statefulSet, 需要警告和重试
		msg := fmt.Sprintf(meta.ResourceOutControlError, "statefulset", namespace, statefulSet.Name, namespace, name)
		meta.Event(namespace, tserver, k8sCoreV1.EventTypeWarning, meta.ResourceOutControlReason, msg)
		return reconclie.RateLimit
	}

	volumeClaimTemplates := meta.BuildStatefulSetVolumeClainTemplates(tserver)
	equal, names := diffVolumeClaimTemplate(statefulSet.Spec.VolumeClaimTemplates, volumeClaimTemplates)
	if !equal {
		return r.recreateStatefulset(tserver, names)
	}

	anyChanged := !meta.EqualTServerAndStatefulSet(tserver, statefulSet)
	if anyChanged {
		return r.syncStatefulset(tserver, statefulSet, namespace, name)
	}
	return reconclie.AllOk
}
