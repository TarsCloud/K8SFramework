package reconcile

import k8sWatchV1 "k8s.io/apimachinery/pkg/watch"

type Result uint

const (
	Done       Result = 0
	Retry      Result = 1
	FatalError Result = 2
	AddAfter   Result = 3
)

type Reconcile interface {
	EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{})
	Start(chan struct{})
}
