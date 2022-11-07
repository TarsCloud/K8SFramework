package storage

import (
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sStorageV1 "k8s.io/api/storage/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
)

type Runner struct {
	provision *TLocalProvisioner
	reconcile *Reconciler
}

func (r *Runner) Init() error {
	claimInformer := tarsRuntime.Factories.K8SInformerFactory.Core().V1().PersistentVolumeClaims()
	claimInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { r.enqueueClaim(obj) },
		UpdateFunc: func(oldObj, newObj interface{}) { r.enqueueClaim(newObj) },
		DeleteFunc: func(obj interface{}) {
		}})

	volumeInformer := tarsRuntime.Factories.K8SInformerFactory.Core().V1().PersistentVolumes()
	volumeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { r.enqueueVolume(obj) },
		UpdateFunc: func(oldObj, newObj interface{}) { r.enqueueVolume(newObj) },
		DeleteFunc: func(obj interface{}) {
		}})

	nodeInformer := tarsRuntime.Factories.K8SInformerFactory.Core().V1().Nodes().Informer()
	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { r.enqueueNode(obj) },
		UpdateFunc: func(oldObj, newObj interface{}) { r.enqueueNode(newObj) },
		DeleteFunc: func(obj interface{}) {
		}})

	classInformer := tarsRuntime.Factories.K8SInformerFactory.Storage().V1().StorageClasses().Informer()
	classInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { r.enqueueStorage(obj) },
		UpdateFunc: func(oldObj, newObj interface{}) { r.enqueueStorage(newObj) },
		DeleteFunc: func(obj interface{}) {
		}})

	r.provision = newTLocalProvisioner()
	r.reconcile = NewReconciler(claimInformer.Lister(), volumeInformer.Lister(), r.provision)
	return nil
}

func (r *Runner) enqueueClaim(obj interface{}) {
	claim := obj.(*k8sCoreV1.PersistentVolumeClaim)
	if claim.Spec.StorageClassName == nil || *claim.Spec.StorageClassName != tarsMeta.TStorageClassName {
		return
	}
	if claim.Spec.VolumeName != "" && !r.provision.ProvisionedBy(claim.Spec.VolumeName) {
		return
	}
	r.reconcile.enqueueClaim(claim)
}

func (r *Runner) enqueueVolume(obj interface{}) {
	volume := obj.(*k8sCoreV1.PersistentVolume)
	if volume.Spec.StorageClassName == tarsMeta.TStorageClassName && r.provision.ProvisionedBy(volume.Name) {
		r.reconcile.enqueueVolume(volume)
	}
}

func (r *Runner) enqueueNode(obj interface{}) {
	node := obj.(*k8sCoreV1.Node)
	if node.Name == r.provision.node {
		if k8sMetaV1.HasLabel(node.ObjectMeta, "tars.io/SupportLocalVolume") {
			r.provision.supportLocalVolume = true
		} else {
			r.provision.supportLocalVolume = false
		}
	}
}

func (r *Runner) enqueueStorage(obj interface{}) {
	class := obj.(*k8sStorageV1.StorageClass)
	if class.Name == tarsMeta.TStorageClassName {
		if class.ReclaimPolicy != nil {
			r.provision.reclaimPolicy = *class.ReclaimPolicy
		} else {
			r.provision.reclaimPolicy = k8sCoreV1.PersistentVolumeReclaimRetain
		}
	}
}

func (r *Runner) Start(stopCh chan struct{}) {
	r.reconcile.Start(stopCh)
}

func NewRunner() *Runner {
	return &Runner{}
}
