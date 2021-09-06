package v1alpha2

import (
	"context"
	"encoding/json"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
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

type TTreeReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTTreeReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *TTreeReconciler {
	reconcile := &TTreeReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *TTreeReconciler) processItem() bool {

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
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TTreeReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1alpha2.TServer:
		tserver := resourceObj.(*crdV1alpha2.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	default:
		return
	}
}

func (r *TTreeReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *TTreeReconciler) reconcile(key string) reconclie.ReconcileResult {
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

	if tserver.DeletionTimestamp != nil {
		return reconclie.AllOk
	}

	ttree, err := r.informers.TTreeInformer.Lister().TTrees(namespace).Get(meta.TTreeResourceName)
	if err != nil {
		msg := fmt.Sprintf(meta.ResourceGetError, "ttree", namespace, meta.TTreeResourceName, err.Error())
		utilRuntime.HandleError(fmt.Errorf(msg))
		meta.Event(namespace, tserver, k8sCoreV1.EventTypeWarning, meta.ResourceGetReason, msg)
		return reconclie.RateLimit
	}

	for i := range ttree.Apps {
		if ttree.Apps[i].Name == tserver.Spec.App {
			return reconclie.AllOk
		}
	}

	newTressApp := &crdV1alpha2.TTreeApp{
		Name:         tserver.Spec.App,
		BusinessRef:  "",
		CreatePerson: "",
		CreateTime:   k8sMetaV1.Now(),
		Mark:         "AddByControl",
	}

	bs, _ := json.Marshal(newTressApp)
	patchContent := fmt.Sprintf("[{\"op\":\"add\",\"path\":\"/apps/-\",\"value\":%s}]", bs)
	_, err = r.clients.CrdClient.CrdV1alpha2().TTrees(namespace).Patch(context.TODO(), meta.TTreeResourceName, patchTypes.JSONPatchType, []byte(patchContent), k8sMetaV1.PatchOptions{})
	if err != nil {
		return reconclie.RateLimit
	}

	return reconclie.AllOk
}
