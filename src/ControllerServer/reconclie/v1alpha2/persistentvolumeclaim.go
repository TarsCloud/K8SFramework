package v1alpha2

import (
	"context"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

type PVCReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewPVCReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *PVCReconciler {
	reconcile := &PVCReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *PVCReconciler) processItem() bool {

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

func (r *PVCReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1alpha2.TServer:
		tserver := resourceObj.(*crdV1alpha2.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *k8sCoreV1.PersistentVolumeClaim:
		if resourceEvent == k8sWatchV1.Deleted {
			break
		}
		pvc := resourceObj.(*k8sCoreV1.PersistentVolumeClaim)
		if pvc.Labels != nil {
			app, appExist := pvc.Labels[meta.TServerAppLabel]
			server, serverExist := pvc.Labels[meta.TServerNameLabel]
			if appExist && serverExist {
				key := fmt.Sprintf("%s/%s-%s", pvc.Namespace, strings.ToLower(app), strings.ToLower(server))
				r.workQueue.Add(key)
				return
			}
		}
	}
}

func (r *PVCReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func buildPVCAnnotations(tserver *crdV1alpha2.TServer) map[string]map[string]string {
	var annotations = make(map[string]map[string]string, 0)
	if tserver.Spec.K8S.Mounts != nil {
		for _, mount := range tserver.Spec.K8S.Mounts {
			if mount.Source.TLocalVolume != nil {
				annotations[mount.Name] = map[string]string{
					meta.TLocalVolumeUIDLabel:  mount.Source.TLocalVolume.UID,
					meta.TLocalVolumeGIDLabel:  mount.Source.TLocalVolume.GID,
					meta.TLocalVolumeModeLabel: mount.Source.TLocalVolume.Mode,
				}
			}
		}
	}
	return annotations
}

func (r *PVCReconciler) reconcile(key string) reconclie.ReconcileResult {
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
		return reconclie.AllOk
	}

	if tserver.DeletionTimestamp != nil || tserver.Spec.K8S.DaemonSet {
		return reconclie.AllOk
	}

	annotations := buildPVCAnnotations(tserver)

	for volumeName, volumeProperties := range annotations {
		appRequirement, _ := labels.NewRequirement(meta.TServerAppLabel, "==", []string{tserver.Spec.App})
		serverRequirement, _ := labels.NewRequirement(meta.TServerNameLabel, "==", []string{tserver.Spec.Server})
		localVolumeRequirement, _ := labels.NewRequirement(meta.TLocalVolumeLabel, "==", []string{volumeName})
		labelSelector := labels.NewSelector().Add(*appRequirement, *serverRequirement, *localVolumeRequirement)
		pvcs, err := r.informers.PersistentVolumeClaimInformer.Lister().PersistentVolumeClaims(namespace).List(labelSelector)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceSelectorError, namespace, "persistentVolumeclaims", err.Error()))
			return reconclie.RateLimit
		}

		for _, pvc := range pvcs {
			if pvc.DeletionTimestamp != nil || meta.ContainLabel(pvc.Annotations, volumeProperties) {
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
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceUpdateError, "PersistentVolumeClaims", namespace, pvc.Name, err.Error()))
			return reconclie.RateLimit
		}
	}
	return reconclie.AllOk
}
