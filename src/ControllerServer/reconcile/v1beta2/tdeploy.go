package v1beta2

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
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

type TDeployReconcile struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTDeployReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TDeployReconcile {
	reconciler := &TDeployReconcile{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconciler)
	return reconciler
}

func (r *TDeployReconcile) processItem() bool {

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
	case reconcile.AllOk:
		r.workQueue.Forget(obj)
		return true
	case reconcile.RateLimit:
		r.workQueue.AddRateLimited(obj)
		return true
	case reconcile.FatalError:
		r.workQueue.ShutDown()
		return false
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TDeployReconcile) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1beta2.TDeploy:
		tdeploy := resourceObj.(*crdV1beta2.TDeploy)
		if tdeploy.Deployed == nil || !*tdeploy.Deployed {
			if tdeploy.Approve != nil && tdeploy.Approve.Result {
				key := fmt.Sprintf("%s/%s", tdeploy.Namespace, tdeploy.Name)
				r.workQueue.Add(key)
			}
		}
	default:
		return
	}
}

func (r *TDeployReconcile) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *TDeployReconcile) reconcile(key string) reconcile.Result {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return reconcile.AllOk
	}

	tdeploy, err := r.informers.TDeployInformer.Lister().TDeploys(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceGetError, "tdeploy", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	if tdeploy.Approve == nil || !tdeploy.Approve.Result {
		return reconcile.AllOk
	}

	if tdeploy.Deployed != nil && *tdeploy.Deployed {
		return reconcile.AllOk
	}

	tserverName := fmt.Sprintf("%s-%s", strings.ToLower(tdeploy.Apply.App), strings.ToLower(tdeploy.Apply.Server))

	newTServer := &crdV1beta2.TServer{
		TypeMeta: k8sMetaV1.TypeMeta{},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserverName,
			Namespace: namespace,
			Labels: map[string]string{
				crdMeta.TServerAppLabel:  tdeploy.Apply.App,
				crdMeta.TServerNameLabel: tdeploy.Apply.Server,
				crdMeta.TSubTypeLabel:    string(tdeploy.Apply.SubType),
			},
		},
		Spec: tdeploy.Apply,
	}

	if _, err = r.clients.CrdClient.CrdV1beta2().TServers(namespace).Create(context.TODO(), newTServer, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceCreateError, "tserver", namespace, newTServer.Name, err.Error()))
		return reconcile.RateLimit
	}

	deployed := true
	tdeployCopy := tdeploy.DeepCopy()
	tdeployCopy.Deployed = &deployed
	if _, err = r.clients.CrdClient.CrdV1beta2().TDeploys(namespace).Update(context.TODO(), tdeployCopy, k8sMetaV1.UpdateOptions{}); err != nil {
		utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceUpdateError, "tdeploy", namespace, name, err.Error()))
		return reconcile.RateLimit
	}

	return reconcile.AllOk
}
