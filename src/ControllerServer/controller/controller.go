package controller

import (
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type Result uint

const (
	Done       Result = 0
	Retry      Result = 1
	FatalError Result = 2
	AddAfter   Result = 3
)

type Controller interface {
	EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{})
	Run(chan struct{})
}

func RegistryInformerEventHandle(resourceKind string, resourceInformer cache.SharedInformer, c Controller) {
	eventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.EnqueueResourceEvent(resourceKind, k8sWatchV1.Added, obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldMeta := oldObj.(k8sMetaV1.Object)
			newMeta := newObj.(k8sMetaV1.Object)
			if newMeta.GetResourceVersion() != oldMeta.GetResourceVersion() {
				c.EnqueueResourceEvent(resourceKind, k8sWatchV1.Modified, newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			c.EnqueueResourceEvent(resourceKind, k8sWatchV1.Deleted, obj)
		},
	}
	resourceInformer.AddEventHandler(eventHandler)
}
