package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	reconcileV1alpha2 "tarscontroller/reconclie/v1alpha2"
	"tarscontroller/webhook"
	"time"
)

func main() {

	stopCh := make(chan struct{})

	clients, informers, err := meta.GetControllerContext()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	hook := webhook.New(clients, informers)
	//call webhook.start() before informers.start(),because informers.start() depend on webhook.conversion

	hook.Start(stopCh)
	time.Sleep(time.Second * 1)

	// new reconcile should before call informers.start() => because reconcile should registry into informers
	reconciles := []reconclie.Reconcile{
		reconcileV1alpha2.NewTDeployReconciler(clients, informers, 1),
		reconcileV1alpha2.NewDaemonSetReconciler(clients, informers, 1),
		reconcileV1alpha2.NewTTreeReconciler(clients, informers, 1),
		reconcileV1alpha2.NewServiceReconciler(clients, informers, 1),
		reconcileV1alpha2.NewTExitedPodReconciler(clients, informers, 3),
		reconcileV1alpha2.NewStatefulSetReconciler(clients, informers, 4),
		reconcileV1alpha2.NewTServerReconciler(clients, informers, 3),
		reconcileV1alpha2.NewTEndpointReconciler(clients, informers, 4),
		reconcileV1alpha2.NewTAccountReconciler(clients, informers, 1),
		reconcileV1alpha2.NewTConfigReconciler(clients, informers, 1),
		reconcileV1alpha2.NewTImageReconciler(clients, informers, 1),
	}

	informers.Start(stopCh)
	if !informers.WaitForCacheSync(stopCh) {
		fmt.Println("WaitForCacheSync Error")
		return
	}

	for _, reconcile := range reconciles {
		reconcile.Start(stopCh)
	}

	wait.Until(func() {
		time.Sleep(time.Second * 1)
	}, 5, stopCh)
}
