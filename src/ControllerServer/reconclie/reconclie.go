package reconclie

import k8sWatchV1 "k8s.io/apimachinery/pkg/watch"

type ReconcileResult uint

const (
	AllOk      ReconcileResult = 0
	RateLimit  ReconcileResult = 1
	FatalError ReconcileResult = 2
	AddAfter   ReconcileResult = 3
)

type Reconcile interface {
	EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{})
	Start(chan struct{})
}
