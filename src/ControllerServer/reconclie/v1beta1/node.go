package v1beta1

import (
	"context"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
	"strings"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	"time"
)

type NodeReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewNodeReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *NodeReconciler {
	reconcile := &NodeReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *NodeReconciler) processItem() bool {

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
	case reconclie.AddAfter:
		r.workQueue.AddAfter(obj, time.Second*1)
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

func (r *NodeReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *k8sCoreV1.Node:
		node := resourceObj.(*k8sCoreV1.Node)
		key := node.Name
		r.workQueue.Add(key)
	default:
		return
	}
}

func (r *NodeReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *NodeReconciler) reconcile(key string) reconclie.ReconcileResult {
	name := key
	node, err := r.informers.NodeInformer.Lister().Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "node", "", name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	if node.DeletionTimestamp != nil || node.Labels == nil {
		return reconclie.AllOk
	}

	nodeNamespaceLabelExist := false
	for k := range node.Labels {
		if strings.HasPrefix(k, meta.TarsNodeLabel+".") {
			nodeNamespaceLabelExist = true
			break
		}
	}

	_, nodeLabelExist := node.Labels[meta.TarsNodeLabel]

	if nodeLabelExist == nodeNamespaceLabelExist {
		return reconclie.AllOk
	}

	nodeCopy := node.DeepCopy()
	if nodeNamespaceLabelExist {
		nodeCopy.Labels[meta.TarsNodeLabel] = ""
	} else {
		delete(nodeCopy.Labels, meta.TarsNodeLabel)
	}

	nodeInterface := r.clients.K8sClient.CoreV1().Nodes()
	if _, err = nodeInterface.Update(context.TODO(), nodeCopy, k8sMetaV1.UpdateOptions{}); err != nil {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceUpdateError, "node", "", name, err.Error()))
		return reconclie.RateLimit
	}

	return reconclie.AllOk
}
