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
	k8sCoreTypeV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sCoreListerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
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

func containLabel(l, r map[string]string) bool {
	if len(l) > len(r) {
		return false
	}

	for lk, lv := range l {
		if rv, ok := r[lk]; !ok || rv != lv {
			return false
		}
	}

	return true
}

type PVCReconciler struct {
	pvcLister k8sCoreListerV1.PersistentVolumeClaimLister
	tsLister  tarsListerV1beta3.TServerLister
	threads   int
	queue     workqueue.RateLimitingInterface
	synced    []cache.InformerSynced
}

func NewPVCController(threads int) *PVCReconciler {
	pvcInformer := tarsRuntime.Factories.K8SInformerFactoryWithTarsFilter.Core().V1().PersistentVolumeClaims()
	tsInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TServers()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8sCoreTypeV1.EventSinkImpl{Interface: tarsRuntime.Clients.K8sClient.CoreV1().Events("")})
	c := &PVCReconciler{
		pvcLister: pvcInformer.Lister(),
		tsLister:  tsInformer.Lister(),
		threads:   threads,
		queue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:    []cache.InformerSynced{pvcInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
	}
	controller.RegistryInformerEventHandle(tarsMeta.KPersistentVolumeClaimKind, tsInformer.Informer(), c)
	controller.RegistryInformerEventHandle(tarsMeta.TServerKind, tsInformer.Informer(), c)
	return c
}

func (r *PVCReconciler) processItem() bool {

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

func (r *PVCReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsV1beta3.TServer:
		tserver := resourceObj.(*tarsV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.queue.Add(key)
	case *k8sCoreV1.PersistentVolumeClaim:
		if resourceEvent == k8sWatchV1.Deleted {
			break
		}
		pvc := resourceObj.(*k8sCoreV1.PersistentVolumeClaim)
		if pvc.Labels != nil {
			app, appExist := pvc.Labels[tarsMeta.TServerAppLabel]
			server, serverExist := pvc.Labels[tarsMeta.TServerNameLabel]
			if appExist && serverExist {
				key := fmt.Sprintf("%s/%s-%s", pvc.Namespace, strings.ToLower(app), strings.ToLower(server))
				r.queue.Add(key)
				return
			}
		}
	}
}

func (r *PVCReconciler) Run(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("persistentvolumeclaim controller", stopCh, r.synced...) {
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

func buildPVCAnnotations(tserver *tarsV1beta3.TServer) map[string]map[string]string {
	var annotations = make(map[string]map[string]string, 0)
	if tserver.Spec.K8S.Mounts != nil {
		for _, mount := range tserver.Spec.K8S.Mounts {
			if mount.Source.TLocalVolume != nil {
				annotations[mount.Name] = map[string]string{
					tarsMeta.TLocalVolumeUIDAnnotation: mount.Source.TLocalVolume.UID,
					tarsMeta.TLocalVolumeGIDAnnotation: mount.Source.TLocalVolume.GID,
					tarsMeta.TLocalVolumeLabel:         mount.Source.TLocalVolume.Mode,
				}
			}
		}
	}
	return annotations
}

func (r *PVCReconciler) reconcile(key string) controller.Result {
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

	if tserver.DeletionTimestamp != nil || tserver.Spec.K8S.DaemonSet {
		return controller.Done
	}

	expectedAnnotations := buildPVCAnnotations(tserver)
	retry := false

	for volumeName, expectedAnnotation := range expectedAnnotations {
		appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tserver.Spec.App})
		serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{tserver.Spec.Server})
		localVolumeRequirement, _ := labels.NewRequirement(tarsMeta.TLocalVolumeLabel, selection.DoubleEquals, []string{volumeName})
		labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*localVolumeRequirement)
		pvcs, err := r.pvcLister.PersistentVolumeClaims(namespace).List(labelSelector)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			klog.Errorf(tarsMeta.ResourceSelectorError, namespace, "persistentVolumeclaims", err.Error())
			return controller.Retry
		}

		for _, pvc := range pvcs {
			if pvc.DeletionTimestamp != nil || containLabel(pvc.Annotations, expectedAnnotation) {
				continue
			}
			pvcCopy := pvc.DeepCopy()
			if pvcCopy.Annotations != nil {
				for k, v := range expectedAnnotation {
					pvcCopy.Annotations[k] = v
				}
			} else {
				pvcCopy.Annotations = expectedAnnotation
			}
			_, err = tarsRuntime.Clients.K8sClient.CoreV1().PersistentVolumeClaims(namespace).Update(context.TODO(), pvcCopy, k8sMetaV1.UpdateOptions{})
			if err == nil {
				continue
			}
			retry = true
			klog.Errorf(tarsMeta.ResourceUpdateError, "PersistentVolumeClaims", namespace, pvc.Name, err.Error())
		}
	}
	if retry {
		return controller.Retry
	}
	return controller.Done
}
