package v1alpha2

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	crdV1alpha2 "k8s.tars.io/api/crd/v1alpha2"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	"time"
)

type TAccountReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func (r *TAccountReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1alpha2.TAccount:
		taccount := resourceObj.(*crdV1alpha2.TAccount)
		key := fmt.Sprintf("%s/%s", taccount.Namespace, taccount.Name)
		r.workQueue.Add(key)
	default:
		return
	}
}

func (r *TAccountReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func NewTAccountReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *TAccountReconciler {
	reconcile := &TAccountReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *TAccountReconciler) processItem() bool {

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

	res, duration := r.reconcile(key)
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
		r.workQueue.Forget(obj)
		if duration != nil {
			r.workQueue.AddAfter(obj, *duration)
		}
		return true
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TAccountReconciler) reconcile(key string) (reconclie.ReconcileResult, *time.Duration) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return reconclie.AllOk, nil
	}

	taccount, err := r.informers.TAccountInformer.Lister().TAccounts(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconclie.AllOk, nil
		}
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "taccount", namespace, name, err.Error()))
		return reconclie.RateLimit, nil
	}

	if taccount.Spec.Authentication.Tokens == nil || len(taccount.Spec.Authentication.Tokens) == 0 {
		return reconclie.AllOk, nil
	}

	currentTime := k8sMetaV1.Now()
	var minDuration *time.Duration

	shouldUpdate := false
	newTokens := make([]*crdV1alpha2.TAccountAuthenticationToken, 0, len(taccount.Spec.Authentication.Tokens))

	for _, token := range taccount.Spec.Authentication.Tokens {
		duration := token.ExpirationTime.Time.Sub(currentTime.Time)
		if duration.Nanoseconds() >= 0 {
			newTokens = append(newTokens, token)
			if minDuration == nil {
				minDuration = &duration
			} else {
				if duration.Nanoseconds() < minDuration.Nanoseconds() {
					minDuration = &duration
				}
			}
		} else {
			shouldUpdate = true
		}
	}

	if shouldUpdate {
		newTaccount := taccount.DeepCopy()
		newTaccount.Spec.Authentication.Tokens = newTokens
		_, err = r.clients.CrdClient.CrdV1alpha2().TAccounts(namespace).Update(context.TODO(), newTaccount, k8sMetaV1.UpdateOptions{})
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourcePatchError, "taccount", namespace, name, err.Error()))
			return reconclie.RateLimit, nil
		}
	}

	return reconclie.AddAfter, minDuration
}
