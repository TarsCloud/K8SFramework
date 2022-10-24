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
	tarsCrdListerV1beta3 "k8s.tars.io/client-go/listers/crd/v1beta3"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/util"
	"time"
)

type PVCReconciler struct {
	clients   *util.Clients
	pvcLister k8sCoreListerV1.PersistentVolumeClaimLister
	tsLister  tarsCrdListerV1beta3.TServerLister
	threads   int
	queue     workqueue.RateLimitingInterface
	synced    []cache.InformerSynced
}

func NewPVCController(clients *util.Clients, factories *util.InformerFactories, threads int) *PVCReconciler {
	pvcInformer := factories.K8SInformerFactoryWithTarsFilter.Core().V1().PersistentVolumeClaims()
	tsInformer := factories.TarsInformerFactory.Crd().V1beta3().TServers()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8sCoreTypeV1.EventSinkImpl{Interface: clients.K8sClient.CoreV1().Events("")})
	c := &PVCReconciler{
		clients:   clients,
		pvcLister: pvcInformer.Lister(),
		tsLister:  tsInformer.Lister(),
		threads:   threads,
		queue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:    []cache.InformerSynced{pvcInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
	}
	controller.SetInformerHandlerEvent(tarsMeta.KPersistentVolumeClaimKind, tsInformer.Informer(), c)
	controller.SetInformerHandlerEvent(tarsMeta.TServerKind, tsInformer.Informer(), c)
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
		utilRuntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
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
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *PVCReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
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

func buildPVCAnnotations(tserver *tarsCrdV1beta3.TServer) map[string]map[string]string {
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
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return controller.Done
	}

	tserver, err := r.tsLister.TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	if tserver.DeletionTimestamp != nil || tserver.Spec.K8S.DaemonSet {
		return controller.Done
	}

	annotations := buildPVCAnnotations(tserver)
	retry := false

	for volumeName, volumeProperties := range annotations {
		appRequirement, _ := labels.NewRequirement(tarsMeta.TServerAppLabel, selection.DoubleEquals, []string{tserver.Spec.App})
		serverRequirement, _ := labels.NewRequirement(tarsMeta.TServerNameLabel, selection.DoubleEquals, []string{tserver.Spec.Server})
		localVolumeRequirement, _ := labels.NewRequirement(tarsMeta.TLocalVolumeLabel, selection.DoubleEquals, []string{volumeName})
		labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*localVolumeRequirement)
		pvcs, err := r.pvcLister.PersistentVolumeClaims(namespace).List(labelSelector)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceSelectorError, namespace, "persistentVolumeclaims", err.Error()))
			return controller.Retry
		}

		for _, pvc := range pvcs {
			if pvc.DeletionTimestamp != nil || ContainLabel(pvc.Annotations, volumeProperties) {
				continue
			}
			pvcCopy := pvc.DeepCopy()
			if pvcCopy.Annotations != nil {
				for k, v := range volumeProperties {
					pvcCopy.Annotations[k] = v
				}
			} else {
				pvcCopy.Annotations = volumeProperties
			}
			_, err := r.clients.K8sClient.CoreV1().PersistentVolumeClaims(namespace).Update(context.TODO(), pvcCopy, k8sMetaV1.UpdateOptions{})
			if err == nil {
				continue
			}
			retry = true
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "PersistentVolumeClaims", namespace, pvc.Name, err.Error()))
		}
	}
	if retry {
		return controller.Retry
	}
	return controller.Done
}
