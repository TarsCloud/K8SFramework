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
	"strings"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	"time"
)

type TDeployReconcile struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTDeployReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *TDeployReconcile {
	reconcile := &TDeployReconcile{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
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

func (r *TDeployReconcile) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1alpha2.TDeploy:
		tdeploy := resourceObj.(*crdV1alpha2.TDeploy)
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

func (r *TDeployReconcile) reconcile(key string) reconclie.ReconcileResult {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid key: %s", key))
		return reconclie.AllOk
	}

	tdeploy, err := r.informers.TDeployInformer.Lister().TDeploys(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "tdeploy", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	if tdeploy.Approve == nil || !tdeploy.Approve.Result {
		return reconclie.AllOk
	}

	if tdeploy.Deployed != nil && *tdeploy.Deployed {
		return reconclie.AllOk
	}

	tserverName := fmt.Sprintf("%s-%s", strings.ToLower(tdeploy.Apply.App), strings.ToLower(tdeploy.Apply.Server))

	newTServer := &crdV1alpha2.TServer{
		TypeMeta: k8sMetaV1.TypeMeta{},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserverName,
			Namespace: namespace,
			Labels: map[string]string{
				meta.TServerAppLabel:  tdeploy.Apply.App,
				meta.TServerNameLabel: tdeploy.Apply.Server,
				meta.TSubTypeLabel:    string(tdeploy.Apply.SubType),
			},
		},
		Spec: tdeploy.Apply,
	}

	if _, err = r.clients.CrdClient.CrdV1alpha2().TServers(namespace).Create(context.TODO(), newTServer, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceCreateError, "tserver", namespace, newTServer.Name, err.Error()))
		return reconclie.RateLimit
	}

	deployed := true
	tdeployCopy := tdeploy.DeepCopy()
	tdeployCopy.Deployed = &deployed
	if _, err := r.clients.CrdClient.CrdV1alpha2().TDeploys(namespace).Update(context.TODO(), tdeployCopy, k8sMetaV1.UpdateOptions{}); err != nil {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceUpdateError, "tdeploy", namespace, name, err.Error()))
		return reconclie.RateLimit
	}

	return reconclie.AllOk
}
