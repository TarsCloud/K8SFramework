package v1beta3

import (
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	"tarscontroller/controller"
)

type TFrameworkConfigReconciler struct {
}

func NewTFrameworkConfigReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TFrameworkConfigReconciler {
	reconciler := &TFrameworkConfigReconciler{}
	informers.Register(reconciler)
	return reconciler
}

func (r *TFrameworkConfigReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TFrameworkConfig:
		tfc := resourceObj.(*tarsCrdV1beta3.TFrameworkConfig)
		if tfc.DeletionTimestamp == nil && resourceEvent != k8sWatchV1.Deleted {
			controller.SetTFrameworkConfig(tfc.Namespace, tfc)
		}
	default:
		return
	}
}

func (r *TFrameworkConfigReconciler) Start(stopCh chan struct{}) {
}
