package v1beta3

import (
	"context"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	k8sCoreListerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"strings"
	"tarscontroller/controller"
	"time"
)

type NodeReconciler struct {
	nodeLister k8sCoreListerV1.NodeLister
	threads    int
	queue      workqueue.RateLimitingInterface
	synced     []cache.InformerSynced
}

func NewNodeController(threads int) *NodeReconciler {
	nodeInformer := tarsRuntime.Factories.K8SInformerFactory.Core().V1().Nodes()
	c := &NodeReconciler{
		nodeLister: nodeInformer.Lister(),
		threads:    threads,
		queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		synced:     []cache.InformerSynced{nodeInformer.Informer().HasSynced},
	}
	controller.RegistryInformerEventHandle(tarsMeta.KNodeKind, nodeInformer.Informer(), c)
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
	case controller.AddAfter:
		r.queue.AddAfter(obj, time.Second*1)
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

func (r *NodeReconciler) Run(stopCh chan struct{}) {
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

func (r *NodeReconciler) reconcile(key string) controller.Result {
	name := key
	node, err := r.nodeLister.Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceGetError, "node", "", name, err.Error())
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

	nodeInterface := tarsRuntime.Clients.K8sClient.CoreV1().Nodes()
	if _, err = nodeInterface.Update(context.TODO(), nodeCopy, k8sMetaV1.UpdateOptions{}); err != nil {
		klog.Errorf(tarsMeta.ResourceUpdateError, "node", "", name, err.Error())
		return controller.Retry
	}

	return controller.Done
}
