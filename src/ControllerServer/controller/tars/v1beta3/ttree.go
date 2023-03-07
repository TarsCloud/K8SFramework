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
	"k8s.io/klog/v2"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsListerV1beta3 "k8s.tars.io/client-go/listers/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"tarscontroller/controller"
	"time"
)

type TTreeReconciler struct {
	trLister tarsListerV1beta3.TTreeLister
	threads  int
	queue    workqueue.RateLimitingInterface
	synced   []cache.InformerSynced
}

func NewTTreeController(threads int) *TTreeReconciler {
	trInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TTrees()
	tsInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TServers()
	c := &TTreeReconciler{
		trLister: trInformer.Lister(),
		threads:  threads,
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:   []cache.InformerSynced{trInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
	}
	controller.RegistryInformerEventHandle(tarsMeta.TTreeKind, trInformer.Informer(), c)
	controller.RegistryInformerEventHandle(tarsMeta.TServerKind, tsInformer.Informer(), c)
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
	default:
		//code should not reach here
		klog.Errorf("should not reach place")
		return false
	}
}

func (r *TTreeReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsV1beta3.TServer:
		tserver := resourceObj.(*tarsV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Spec.App)
		r.queue.Add(key)
	default:
		return
	}
}

func (r *TTreeReconciler) Run(stopCh chan struct{}) {
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

func (r *TTreeReconciler) reconcile(key string) controller.Result {
	namespace, app, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		klog.Errorf("invalid key: %s", key)
		return controller.Done
	}

	ttree, err := r.trLister.TTrees(namespace).Get(tarsMeta.FixedTTreeResourceName)
	if err != nil {
		msg := fmt.Sprintf(tarsMeta.ResourceGetError, "ttree", namespace, tarsMeta.FixedTTreeResourceName, err.Error())
		klog.Errorf(msg)
		return controller.Retry
	}

	for i := range ttree.Apps {
		if ttree.Apps[i].Name == app {
			return controller.Done
		}
	}

	newTressApp := &tarsV1beta3.TTreeApp{
		Name:         app,
		BusinessRef:  "",
		CreatePerson: "",
		CreateTime:   k8sMetaV1.Now(),
		Mark:         "AddByController",
	}
	jsonPatch := tarsTool.JsonPatch{
		{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/apps/-",
			Value: newTressApp,
		},
	}

	patchContent, _ := json.Marshal(jsonPatch)
	_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TTrees(namespace).Patch(context.TODO(), tarsMeta.FixedTTreeResourceName, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
	if err != nil {
		return controller.Retry
	}

	return controller.Done
}
