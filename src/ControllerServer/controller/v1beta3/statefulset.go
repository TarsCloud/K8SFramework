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
	tarsCrdListerV1beta3 "k8s.tars.io/client-go/listers/crd/v1beta3"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/controller"
	"tarscontroller/util"
	"time"
)

type StatefulSetReconciler struct {
	clients       *util.Clients
	stsLister     k8sAppsListerV1.StatefulSetLister
	tsLister      tarsCrdListerV1beta3.TServerLister
	threads       int
	queue         workqueue.RateLimitingInterface
	synced        []cache.InformerSynced
	eventRecorder record.EventRecorder
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

func NewStatefulSetController(clients *util.Clients, factories *util.InformerFactories, threads int) *StatefulSetReconciler {
	stsInformer := factories.K8SInformerFactoryWithTarsFilter.Apps().V1().StatefulSets()
	tsInformer := factories.TarsInformerFactory.Crd().V1beta3().TServers()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8sCoreTypeV1.EventSinkImpl{Interface: clients.K8sClient.CoreV1().Events("")})
	sc := &StatefulSetReconciler{
		clients:       clients,
		stsLister:     stsInformer.Lister(),
		tsLister:      tsInformer.Lister(),
		threads:       threads,
		queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:        []cache.InformerSynced{stsInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
		eventRecorder: eventBroadcaster.NewRecorder(scheme.Scheme, k8sCoreV1.EventSource{Component: "daemonset-controller"}),
	}
	controller.SetInformerHandlerEvent(tarsMeta.KStatefulSetKind, stsInformer.Informer(), sc)
	controller.SetInformerHandlerEvent(tarsMeta.TServerKind, tsInformer.Informer(), sc)
	return sc
}

func (r *StatefulSetReconciler) processItem() bool {

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

	res := r.sync(key)

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
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *StatefulSetReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.queue.Add(key)
	case *k8sAppsV1.StatefulSet:
		statefulset := resourceObj.(*k8sAppsV1.StatefulSet)
		if statefulset.DeletionTimestamp != nil || resourceEvent == k8sWatchV1.Deleted {
			key := fmt.Sprintf("%s/%s", statefulset.Namespace, statefulset.Name)
			r.queue.Add(key)
		}
	}
}

func (r *StatefulSetReconciler) StartController(stopCh chan struct{}) {
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

func (r *StatefulSetReconciler) syncStatefulset(tserver *tarsCrdV1beta3.TServer, statefulSet *k8sAppsV1.StatefulSet, namespace, name string) controller.Result {
	statefulSetCopy := statefulSet.DeepCopy()
	syncStatefulSet(tserver, statefulSetCopy)
	statefulSetInterface := r.clients.K8sClient.AppsV1().StatefulSets(namespace)
	if _, err := statefulSetInterface.Update(context.TODO(), statefulSetCopy, k8sMetaV1.UpdateOptions{}); err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "statefulset", namespace, name, err.Error()))
		return controller.Retry
	}
	return controller.Done
}

func (r *StatefulSetReconciler) recreateStatefulset(tserver *tarsCrdV1beta3.TServer, shouldDeletePVCS []string) controller.Result {
	namespace := tserver.Namespace
	name := tserver.Name
	err := r.clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "statefulset", namespace, name, err.Error()))
		return controller.Retry
	}

	if shouldDeletePVCS != nil {
		appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tserver.Spec.App})
		serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{tserver.Spec.Server})
		labelSelector := labels.NewSelector().Add(*appRequirement, *serverRequirement)

		localVolumeRequirement, _ := labels.NewRequirement(tarsMeta.TLocalVolumeLabel, selection.DoubleEquals, shouldDeletePVCS)
		labelSelector.Add(*localVolumeRequirement)

		err = r.clients.K8sClient.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(context.TODO(), k8sMetaV1.DeleteOptions{}, k8sMetaV1.ListOptions{
			LabelSelector: labelSelector.String(),
		})

		if err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteCollectionError, "persistentvolumeclaims", labelSelector.String(), err.Error()))
		}
	}
	return controller.AddAfter
}

func (r *StatefulSetReconciler) sync(key string) controller.Result {
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
		err = r.clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "statefulset", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	if tserver.DeletionTimestamp != nil || tserver.Spec.K8S.DaemonSet {
		err = r.clients.K8sClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceDeleteError, "statefulset", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	statefulSet, err := r.stsLister.StatefulSets(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "statefulset", namespace, name, err.Error()))
			return controller.Retry
		}

		if !tserver.Spec.K8S.DaemonSet {
			statefulSet = buildStatefulset(tserver)
			statefulSetInterface := r.clients.K8sClient.AppsV1().StatefulSets(namespace)
			_, err = statefulSetInterface.Create(context.TODO(), statefulSet, k8sMetaV1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceCreateError, "statefulset", namespace, name, err.Error()))
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

	volumeClaimTemplates := buildStatefulsetVolumeClaimTemplates(tserver)
	equal, names := diffVolumeClaimTemplate(statefulSet.Spec.VolumeClaimTemplates, volumeClaimTemplates)
	if !equal {
		return r.recreateStatefulset(tserver, names)
	}

	anyChanged := !EqualTServerAndStatefulSet(tserver, statefulSet)
	if anyChanged {
		return r.syncStatefulset(tserver, statefulSet, namespace, name)
	}
	return controller.Done
}
