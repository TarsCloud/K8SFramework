package v1beta3

import (
	"context"
	"fmt"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
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

type TTreeReconciler struct {
	clients  *util.Clients
	trLister tarsCrdListerV1beta3.TTreeLister
	threads  int
	queue    workqueue.RateLimitingInterface
	synced   []cache.InformerSynced
}

func NewTTreeController(clients *util.Clients, factories *util.InformerFactories, threads int) *TTreeReconciler {
	trInformer := factories.TarsInformerFactory.Crd().V1beta3().TTrees()
	tsInformer := factories.TarsInformerFactory.Crd().V1beta3().TServers()
	c := &TTreeReconciler{
		clients:  clients,
		trLister: trInformer.Lister(),
		threads:  threads,
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:   []cache.InformerSynced{trInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
	}
	controller.SetInformerHandlerEvent(tarsMeta.TTreeKind, trInformer.Informer(), c)
	controller.SetInformerHandlerEvent(tarsMeta.TServerKind, tsInformer.Informer(), c)
	return c
}

func (r *TTreeReconciler) processItem() bool {

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
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TTreeReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Spec.App)
		r.queue.Add(key)
	default:
		return
	}
}

func (r *TTreeReconciler) StartController(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("ttree controller", stopCh, r.synced...) {
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

func (r *TTreeReconciler) sync(key string) controller.Result {
	namespace, app, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return controller.Done
	}

	ttree, err := r.trLister.TTrees(namespace).Get(tarsMeta.FixedTTreeResourceName)
	if err != nil {
		msg := fmt.Sprintf(tarsMeta.ResourceGetError, "ttree", namespace, tarsMeta.FixedTTreeResourceName, err.Error())
		utilRuntime.HandleError(fmt.Errorf(msg))
		return controller.Retry
	}

	for i := range ttree.Apps {
		if ttree.Apps[i].Name == app {
			return controller.Done
		}
	}

	newTressApp := &tarsCrdV1beta3.TTreeApp{
		Name:         app,
		BusinessRef:  "",
		CreatePerson: "",
		CreateTime:   k8sMetaV1.Now(),
		Mark:         "AddByController",
	}
	jsonPatch := tarsMeta.JsonPatch{
		{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/apps/-",
			Value: newTressApp,
		},
	}

	patchContent, _ := json.Marshal(jsonPatch)
	_, err = r.clients.CrdClient.CrdV1beta3().TTrees(namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
	if err != nil {
		return controller.Retry
	}

	return controller.Done
}
