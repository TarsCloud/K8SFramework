package storage

import (
	"context"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sCoreListerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"time"
)

type Result int

const (
	AllOk      Result = 0
	RateLimit  Result = 1
	FatalError Result = 2
	AddAfter   Result = 3
)

type Reconciler struct {
	claimLister  k8sCoreListerV1.PersistentVolumeClaimLister
	volumeLister k8sCoreListerV1.PersistentVolumeLister

	provision *TLocalProvisioner

	claimQueue  workqueue.RateLimitingInterface
	volumeQueue workqueue.RateLimitingInterface

	eventRecorder record.EventRecorder
}

func (r *Reconciler) enqueueVolume(volume *k8sCoreV1.PersistentVolume) {
	r.volumeQueue.Add(volume.Name)
}

func (r *Reconciler) enqueueClaim(claim *k8sCoreV1.PersistentVolumeClaim) {
	key := fmt.Sprintf("%s/%s", claim.Namespace, claim.Name)
	r.claimQueue.Add(key)
}

func (r *Reconciler) processItem(queue workqueue.RateLimitingInterface, reconcile func(key string) (Result, *time.Duration)) bool {
	obj, shutdown := queue.Get()
	if shutdown {
		return false
	}
	defer queue.Done(obj)
	key, ok := obj.(string)
	if !ok {
		klog.Errorf("expected string in queue but got %#v", obj)
		queue.Forget(key)
		return true
	}
	res, duration := reconcile(key)
	switch res {
	case AllOk:
		queue.Forget(obj)
		return true
	case RateLimit:
		queue.AddRateLimited(obj)
		return true
	case AddAfter:
		queue.AddAfter(obj, *duration)
		return true
	case FatalError:
		queue.ShutDown()
		return false
	default:
		//code should not reach here
		klog.Errorf("should not reach place")
		return false
	}
}

func (r *Reconciler) reconcileClaim(key string) (Result, *time.Duration) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		klog.Errorf("observed unexpected claim key: %s, skip", key)
		return AllOk, nil
	}

	claim, err := r.claimLister.PersistentVolumeClaims(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("observed claim(%s) deleted", key)
			return AllOk, nil
		}
		return RateLimit, nil
	}

	volumeName := claim.Spec.VolumeName

	if claim.DeletionTimestamp != nil {
		klog.Infof("observed claim(%s) terminating", key)

		if volumeName == "" {
			klog.Infof("observed the volumeName of terminating claim(%s) is empty, skip", key)
			return AllOk, nil
		}

		if !tarsTool.HasFinalizer(claim.Finalizers, PVCProtectionFinalizer) {
			klog.Infof("observed the finalizer of terminating claim(%s) is empty, skip", key)
			return AllOk, nil
		}

		volume, err := r.volumeLister.Get(volumeName)
		if err != nil {
			if !errors.IsNotFound(err) {
				klog.Infof("get the volume(%s) bound to terminating claim(%s) failed: %s", volumeName, key, err.Error())
				return RateLimit, nil
			}

			klog.Infof("observed the volume(%s) bound to terminating claim(%s) released", volumeName, key)
			klog.Infof("begin to remove finalizer(%s) for terminating claim(%s)", PVCProtectionFinalizer, key)
			newClaim := claim.DeepCopy()
			newClaim.Finalizers = tarsTool.RemoveFinalizer(claim.Finalizers, PVCProtectionFinalizer)
			_, err = tarsRuntime.Clients.K8sClient.CoreV1().PersistentVolumeClaims(namespace).Update(context.TODO(), newClaim, k8sMetaV1.UpdateOptions{})

			if err == nil {
				klog.Infof("remove finalizer(%s) for terminating claim(%s) success", PVCProtectionFinalizer, key)
				return AllOk, nil
			}

			if errors.IsNotFound(err) {
				klog.Infof("remove finalizer(%s) for terminating claim(%s) failed because claim has been released early, skip", PVCProtectionFinalizer, key)
				return AllOk, nil
			}

			klog.Errorf("remove finalizer(%s) for terminating claim(%s) failed: %s", PVCProtectionFinalizer, key, err.Error())
			return RateLimit, nil
		}

		if volume.DeletionTimestamp == nil {
			klog.Infof("begin to delete volume(%s) for terminating claim(%s)", volumeName, key)
			_ = tarsRuntime.Clients.K8sClient.CoreV1().PersistentVolumes().Delete(context.TODO(), volumeName, k8sMetaV1.DeleteOptions{})
		} else {
			klog.Infof("observed the volume(%s) bound to terminating claim(%s) already in terminating", volumeName, key)
			r.volumeQueue.Add(volumeName)
		}
		return RateLimit, nil
	}

	shouldProvision := false
	if claim.Status.Phase == k8sCoreV1.ClaimPending {
		klog.Infof("observed claim(%s) pending", key)
		volumeName, err = r.provision.VolumeName(claim)
		if err != nil {
			klog.Infof("generate volume name for claim(%s) failed: %s, skip", key, err.Error())
			return AllOk, nil
		}

		_, err = r.volumeLister.Get(volumeName)
		if err == nil {
			/*
				Imagine that we have a group of claims [ claim-0, claim-1, claim-2], and claim-0 has bound a volume,
				Then, we release claim-0.

				If [ claim-1, claim-2 ] not in the work queue at this time,
				we have no chance to provide the volume for [ claim-1, claim-2 ]
			*/
			duration := time.Minute
			return AddAfter, &duration
		}

		if !errors.IsNotFound(err) {
			klog.Infof("get the volume(%s) for pending claim(%s) failed: %s, try times(%d)", volumeName, key, err.Error(), r.claimQueue.NumRequeues(key))
			return RateLimit, nil
		}
		shouldProvision = true
	}

	if claim.Status.Phase == k8sCoreV1.ClaimLost {
		klog.Infof("observed claim(%s) lost", key)
		shouldProvision = true
	}

	if shouldProvision {
		klog.Infof("begin to provision volume(%s) for claim(%s)", volumeName, key)
		volume, state, err := r.provision.Provision(claim)
		if state == ProvisioningAgain {
			klog.Errorf("provision volume(%s) for claim(%s) failed: %s, try times(%d)", volumeName, key, err.Error(), r.claimQueue.NumRequeues(key))
			return RateLimit, nil
		}

		if err != nil {
			klog.Errorf("provision volume(%s) for claim(%s) failed: %s, skip", volumeName, key, err.Error())
			return AllOk, nil
		}

		klog.Infof("provision volume(%s) for claim(%s) success", volumeName, key)

		klog.Infof("begin to create persistentvolumes(%s) for claim(%s)", volumeName, key)
		_, err = tarsRuntime.Clients.K8sClient.CoreV1().PersistentVolumes().Create(context.TODO(), volume, k8sMetaV1.CreateOptions{})

		if err != nil {
			if errors.IsAlreadyExists(err) {
				klog.Infof("create persistentvolumes(%s) for claim(%s) failed because persistentvolumes has been create early", volumeName, key)
				return AllOk, nil
			}
			klog.Infof("create persistentvolumes(%s) for claim(%s) failed: %s, try times(%d)", volumeName, key, err.Error(), r.claimQueue.NumRequeues(key))
			return RateLimit, nil
		}
		klog.Infof("create persistentvolumes(%s) for claim(%s) success", volumeName, key)
		return AllOk, nil
	}

	if claim.Status.Phase == k8sCoreV1.ClaimBound {
		/*
			gid,uid,mode may be modified bypassing tserver,
			delaying sync operation avoids changing gid|uid|perm twice
		*/
		counts := r.claimQueue.NumRequeues(key)
		if counts == 0 {
			klog.Infof("observed volume(%s) bound to claim(%s)", volumeName, key)
		}

		if counts >= 2 {
			state, err := r.provision.SyncClaim(claim)
			if err != nil {
				klog.Errorf("sync volume(%s) for claim(%s) failed: %s, try times(%d)", volumeName, key, err.Error(), counts-2)
			}
			if state == ProvisioningFinished {
				klog.Infof("sync volume(%s) for claim(%s) success", volumeName, key)
				return AllOk, nil
			}
		}

		return RateLimit, nil
	}

	klog.Errorf("should not reach place")
	return AllOk, nil
}

func (r *Reconciler) reconcileVolume(key string) (Result, *time.Duration) {
	name := key
	volume, err := r.volumeLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("observed volume(%s) deleted", key)
			return AllOk, nil
		}
		return RateLimit, nil
	}

	if volume.DeletionTimestamp != nil {
		klog.Infof("observed volume(%s) terminating", key)
		if !tarsTool.HasFinalizer(volume.Finalizers, PVProtectionFinalizer) {
			klog.Infof("observed the finalizer of terminating volume(%s) is empty, skip", key)
			return AllOk, nil
		}

		claimRef := volume.Spec.ClaimRef
		if claimRef != nil {
			claim, err := r.claimLister.PersistentVolumeClaims(claimRef.Namespace).Get(claimRef.Name)
			if err != nil && !errors.IsNotFound(err) {
				klog.Errorf("get the the claim(%s/%s) referenced by volume(%s) failed: %s, try times(%d)", claimRef.Namespace, claimRef.Name, key, err.Error(), r.claimQueue.NumRequeues(key))
				return RateLimit, nil
			}

			if claim != nil && claim.DeletionTimestamp == nil {
				klog.Errorf("observed the claim(%s/%s) referenced by volume(%s) not in terminating, delay", claimRef.Namespace, claimRef.Name, key)
				duration := time.Minute * 1
				return AddAfter, &duration
			}
		}

		klog.Infof("begin to delete directory for terminating volume(%s)", key, key)
		err = r.provision.Delete(volume)
		if err != nil {
			klog.Errorf("delete directory for terminating volume(%s) failed, try times(%d)", err, r.claimQueue.NumRequeues(key))
			return RateLimit, nil
		}
		klog.Errorf("delete directory for terminating volume(%s) success", key)
		newVolume := volume.DeepCopy()
		newVolume.Finalizers = tarsTool.RemoveFinalizer(volume.Finalizers, PVProtectionFinalizer)
		klog.Infof("begin to remove finalizer(%s) for terminating volume(%s)", PVProtectionFinalizer, key)
		_, err = tarsRuntime.Clients.K8sClient.CoreV1().PersistentVolumes().Update(context.TODO(), newVolume, k8sMetaV1.UpdateOptions{})
		if err == nil {
			klog.Infof("remove finalizer(%s) for terminating volume(%s) success", PVProtectionFinalizer, key)
			return AllOk, nil
		}

		if errors.IsNotFound(err) {
			klog.Infof("remove finalizer(%s) for terminating volume(%s) failed because volume has been released early, skip", PVProtectionFinalizer, key)
			return AllOk, nil
		}

		klog.Errorf("remove finalizer(%s) for terminating volume(%s) failed: %s", PVProtectionFinalizer, key, err.Error())
		return RateLimit, nil
	}

	if volume.Status.Phase == k8sCoreV1.VolumeAvailable {
		klog.Infof("observed volume(%s) available", key)
		if volume.CreationTimestamp.Add(24 * time.Hour).Before(time.Now()) {
			klog.Infof("observed volume(%s) idle for over 24 hours, will release it", key)
			_ = tarsRuntime.Clients.K8sClient.CoreV1().PersistentVolumes().Delete(context.TODO(), volume.Name, k8sMetaV1.DeleteOptions{})
		}
		duration := time.Minute * 10
		return AddAfter, &duration
	}
	return AllOk, nil
}

func (r *Reconciler) Start(stopCh chan struct{}) {
	go wait.Until(func() { r.processItem(r.claimQueue, r.reconcileClaim) }, time.Second, stopCh)
	go wait.Until(func() { r.processItem(r.volumeQueue, r.reconcileVolume) }, time.Second, stopCh)
}

func NewReconciler(claimLister k8sCoreListerV1.PersistentVolumeClaimLister, volumeLister k8sCoreListerV1.PersistentVolumeLister, provision *TLocalProvisioner) *Reconciler {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, 30*time.Second)
	return &Reconciler{
		claimLister:  claimLister,
		volumeLister: volumeLister,
		provision:    provision,
		claimQueue:   workqueue.NewRateLimitingQueue(rateLimiter),
		volumeQueue:  workqueue.NewRateLimitingQueue(rateLimiter),
	}
}
