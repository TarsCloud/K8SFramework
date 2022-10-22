package v1beta3

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
	tarsCrdListerV1beta3 "k8s.tars.io/client-go/listers/crd/v1beta3"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"tarscontroller/controller"
	"tarscontroller/util"
	"time"
)

type TAccountReconciler struct {
	clients  *util.Clients
	taLister tarsCrdListerV1beta3.TAccountLister
	threads  int
	queue    workqueue.RateLimitingInterface
	synced   []cache.InformerSynced
}

func (r *TAccountReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TAccount:
		taccount := resourceObj.(*tarsCrdV1beta3.TAccount)
		key := fmt.Sprintf("%s/%s", taccount.Namespace, taccount.Name)
		r.queue.Add(key)
	default:
		return
	}
}

func (r *TAccountReconciler) StartController(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("taccount controller", stopCh, r.synced...) {
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

func NewTAccountController(clients *util.Clients, factories *util.InformerFactories, threads int) *TAccountReconciler {
	taInformer := factories.TarsInformerFactory.Crd().V1beta3().TAccounts()
	tac := &TAccountReconciler{
		clients:  clients,
		taLister: taInformer.Lister(),
		threads:  threads,
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:   []cache.InformerSynced{taInformer.Informer().HasSynced},
	}
	controller.SetInformerHandlerEvent(tarsMeta.TAccountKind, taInformer.Informer(), tac)
	return tac
}

func (r *TAccountReconciler) processItem() bool {

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

	res, duration := r.sync(key)
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
		r.queue.Forget(obj)
		if duration != nil {
			r.queue.AddAfter(obj, *duration)
		}
		return true
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TAccountReconciler) sync(key string) (controller.Result, *time.Duration) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return controller.Done, nil
	}

	taccount, err := r.taLister.TAccounts(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return controller.Done, nil
		}
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "taccount", namespace, name, err.Error()))
		return controller.Retry, nil
	}

	if taccount.Spec.Authentication.Tokens == nil || len(taccount.Spec.Authentication.Tokens) == 0 {
		return controller.Done, nil
	}

	currentTime := k8sMetaV1.Now()
	var minDuration *time.Duration

	shouldUpdate := false
	newTokens := make([]*tarsCrdV1beta3.TAccountAuthenticationToken, 0, len(taccount.Spec.Authentication.Tokens))

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
		_, err = r.clients.CrdClient.CrdV1beta3().TAccounts(namespace).Update(context.TODO(), newTaccount, k8sMetaV1.UpdateOptions{})
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourcePatchError, "taccount", namespace, name, err.Error()))
			return controller.Retry, nil
		}
	}

	return controller.AddAfter, minDuration
}
