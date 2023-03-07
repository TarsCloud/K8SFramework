package v1beta3

import (
	"context"
	"fmt"
	k8sAppsV1 "k8s.io/api/apps/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	k8sCoreTypeV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sAppsListerV1 "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsListerV1beta3 "k8s.tars.io/client-go/listers/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"tarscontroller/controller"
	"time"
)

type StatefulSetReconciler struct {
	stsLister     k8sAppsListerV1.StatefulSetLister
	tsLister      tarsListerV1beta3.TServerLister
	threads       int
	queue         workqueue.RateLimitingInterface
	synced        []cache.InformerSynced
	eventRecorder record.EventRecorder
}

func diffVolumeClaimTemplate(current, target []k8sCoreV1.PersistentVolumeClaim) (bool, []string) {
	if current == nil && target != nil {
		return false, nil
	}

	tcs := make(map[string]*k8sCoreV1.PersistentVolumeClaim, len(target))
	for i := range target {
		tcs[target[i].Name] = &target[i]
	}

	var equal = true
	var shouldDeletes []string

	for i := range current {
		cc := &current[i]
		tc, ok := tcs[cc.Name]
		if !ok && cc.Spec.StorageClassName != nil && *cc.Spec.StorageClassName == tarsMeta.TStorageClassName {
			equal = false
			shouldDeletes = append(shouldDeletes, cc.Name)
			continue
		}

		if equal == true {
			if !equality.Semantic.DeepEqual(cc.ObjectMeta, tc.ObjectMeta) {
				equal = false
				continue
			}
		}
	}
	return equal, shouldDeletes
}

func NewStatefulSetController(threads int) *StatefulSetReconciler {
	stsInformer := tarsRuntime.Factories.K8SInformerFactoryWithTarsFilter.Apps().V1().StatefulSets()
	tsInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TServers()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8sCoreTypeV1.EventSinkImpl{Interface: tarsRuntime.Clients.K8sClient.CoreV1().Events("")})
	c := &StatefulSetReconciler{
		stsLister:     stsInformer.Lister(),
		tsLister:      tsInformer.Lister(),
		threads:       threads,
		queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:        []cache.InformerSynced{stsInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
		eventRecorder: eventBroadcaster.NewRecorder(scheme.Scheme, k8sCoreV1.EventSource{Component: "daemonset-controller"}),
	}
	controller.RegistryInformerEventHandle(tarsMeta.KStatefulSetKind, stsInformer.Informer(), c)
	controller.RegistryInformerEventHandle(tarsMeta.TServerKind, tsInformer.Informer(), c)
	return c
}

func (r *StatefulSetReconciler) processItem() bool {

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
	case controller.AddAfter:
		r.queue.AddAfter(obj, time.Second*3)
		return true
	default:
		//code should not reach here
		klog.Errorf("should not reach place")
		return false
	}
}

func (r *StatefulSetReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsV1beta3.TServer:
		tserver := resourceObj.(*tarsV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.queue.Add(key)
	case *k8sAppsV1.StatefulSet:
		statefulset := resourceObj.(*k8sAppsV1.StatefulSet)
		if resourceEvent == k8sWatchV1.Deleted {
			key := fmt.Sprintf("%s/%s", statefulset.Namespace, statefulset.Name)
			r.queue.Add(key)
		}
	}
}

func (r *StatefulSetReconciler) Run(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("statefulset controller", stopCh, r.synced...) {
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

func (r *StatefulSetReconciler) rebuildStatefulset(tserver *tarsV1beta3.TServer, shouldDeletes []string) controller.Result {
	namespace := tserver.Namespace
	name := tserver.Name
	err := tarsRuntime.Clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf(tarsMeta.ResourceDeleteError, "statefulset", namespace, name, err.Error())
		return controller.Retry
	}

	if shouldDeletes != nil {
		appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tserver.Spec.App})
		serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{tserver.Spec.Server})
		localVolumeRequirement, _ := labels.NewRequirement(tarsMeta.TLocalVolumeLabel, selection.In, shouldDeletes)
		labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*localVolumeRequirement)
		err = tarsRuntime.Clients.K8sClient.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
			LabelSelector: labelSelector.String(),
		})

		if err != nil {
			klog.Errorf(tarsMeta.ResourceDeleteCollectionError, "persistentvolumeclaims", labelSelector.String(), err.Error())
		}
	}
	return controller.AddAfter
}

func (r *StatefulSetReconciler) reconcile(key string) controller.Result {
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
		err = tarsRuntime.Clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceDeleteError, "statefulset", namespace, name, err.Error())
			return controller.Retry
		}
		return controller.Done
	}

	if tserver.DeletionTimestamp != nil || tserver.Spec.K8S.DaemonSet {
		err = tarsRuntime.Clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceDeleteError, "statefulset", namespace, name, err.Error())
			return controller.Retry
		}
		return controller.Done
	}

	statefulSet, err := r.stsLister.StatefulSets(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceGetError, "statefulset", namespace, name, err.Error())
			return controller.Retry
		}

		if !tserver.Spec.K8S.DaemonSet {
			statefulSet = tarsRuntime.TarsTranslator.BuildStatefulset(tserver)
			statefulSetInterface := tarsRuntime.Clients.K8sClient.AppsV1().StatefulSets(namespace)
			_, err = statefulSetInterface.Create(context.TODO(), statefulSet, k8sMetaV1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				klog.Errorf(tarsMeta.ResourceCreateError, "statefulset", namespace, name, err.Error())
				return controller.Retry
			}
		}
		return controller.Done
	}

	if statefulSet.DeletionTimestamp != nil {
		return controller.AddAfter
	}

	if !k8sMetaV1.IsControlledBy(statefulSet, tserver) {
		// 此处意味着出现了非由 controller 管理的同名 statefulSet, 需要警告和重试
		msg := fmt.Sprintf(tarsMeta.ResourceOutControlError, "statefulset", namespace, statefulSet.Name, namespace, name)
		r.eventRecorder.Event(tserver, k8sCoreV1.EventTypeWarning, tarsMeta.ResourceOutControlReason, msg)
		return controller.Retry
	}

	volumeClaimTemplates := tarsRuntime.TarsTranslator.BuildStatefulsetVolumeClaimTemplates(tserver)
	equal, shouldDeletes := diffVolumeClaimTemplate(statefulSet.Spec.VolumeClaimTemplates, volumeClaimTemplates)
	if !equal {
		return r.rebuildStatefulset(tserver, shouldDeletes)
	}

	update, target := tarsRuntime.TarsTranslator.DryRunSyncStatefulset(tserver, statefulSet)
	if update {
		statefulSetInterface := tarsRuntime.Clients.K8sClient.AppsV1().StatefulSets(namespace)
		if _, err = statefulSetInterface.Update(context.TODO(), target, k8sMetaV1.UpdateOptions{}); err != nil {
			klog.Errorf(tarsMeta.ResourceUpdateError, "statefulset", namespace, name, err.Error())
			return controller.Retry
		}
	}
	return controller.Done
}
