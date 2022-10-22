package v1beta3

import (
	"context"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	k8sCoreListerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/util"
	"time"
)

type NodeReconciler struct {
	clients    *util.Clients
	nodeLister k8sCoreListerV1.NodeLister
	threads    int
	queue      workqueue.RateLimitingInterface
	synced     []cache.InformerSynced
}

func NewNodeController(clients *util.Clients, factories *util.InformerFactories, threads int) *NodeReconciler {
	nodeInformer := factories.K8SInformerFactory.Core().V1().Nodes()
	c := &NodeReconciler{
		clients:    clients,
		nodeLister: nodeInformer.Lister(),
		threads:    threads,
		queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		synced:     []cache.InformerSynced{nodeInformer.Informer().HasSynced},
	}
	controller.SetInformerHandlerEvent(tarsMeta.KNodeKind, nodeInformer.Informer(), c)
	return c
}

func (r *NodeReconciler) processItem() bool {

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
	case controller.AddAfter:
		r.queue.AddAfter(obj, time.Second*1)
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

func (r *NodeReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *k8sCoreV1.Node:
		node := resourceObj.(*k8sCoreV1.Node)
		key := node.Name
		r.queue.Add(key)
	default:
		return
	}
}

func (r *NodeReconciler) StartController(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("node controller", stopCh, r.synced...) {
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

func (r *NodeReconciler) sync(key string) controller.Result {
	name := key
	node, err := r.nodeLister.Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "node", "", name, err.Error()))
			return controller.Retry
		}
		return controller.Done
	}

	if node.DeletionTimestamp != nil || node.Labels == nil {
		return controller.Done
	}

	nodeNamespaceLabelExist := false
	for k := range node.Labels {
		if strings.HasPrefix(k, tarsMeta.TarsNodeLabel+".") {
			nodeNamespaceLabelExist = true
			break
		}
	}

	_, nodeLabelExist := node.Labels[tarsMeta.TarsNodeLabel]

	if nodeLabelExist == nodeNamespaceLabelExist {
		return controller.Done
	}

	nodeCopy := node.DeepCopy()
	if nodeNamespaceLabelExist {
		nodeCopy.Labels[tarsMeta.TarsNodeLabel] = ""
	} else {
		delete(nodeCopy.Labels, tarsMeta.TarsNodeLabel)
	}

	nodeInterface := r.clients.K8sClient.CoreV1().Nodes()
	if _, err = nodeInterface.Update(context.TODO(), nodeCopy, k8sMetaV1.UpdateOptions{}); err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceUpdateError, "node", "", name, err.Error()))
		return controller.Retry
	}

	return controller.Done
}
