package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/tools/leaderelection"
	"os"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	reconcileV1beta1 "tarscontroller/reconclie/v1beta1"
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

	hooks := webhook.New(clients, informers)
	//call webhook.start() before informers.start(),because informers.start() depend on conversion webhook service

	hooks.Start(stopCh)
	time.Sleep(time.Second * 1)

	// new reconcile should before call informers.start() => because reconcile should registry into informers
	reconciles := []reconclie.Reconcile{
		reconcileV1beta1.NewNodeReconciler(clients, informers, 1),
		reconcileV1beta1.NewTDeployReconciler(clients, informers, 1),
		reconcileV1beta1.NewDaemonSetReconciler(clients, informers, 1),
		reconcileV1beta1.NewTTreeReconciler(clients, informers, 1),
		reconcileV1beta1.NewServiceReconciler(clients, informers, 1),
		reconcileV1beta1.NewTExitedPodReconciler(clients, informers, 1),
		reconcileV1beta1.NewStatefulSetReconciler(clients, informers, 3),
		reconcileV1beta1.NewTServerReconciler(clients, informers, 1),
		reconcileV1beta1.NewTEndpointReconciler(clients, informers, 3),
		reconcileV1beta1.NewTAccountReconciler(clients, informers, 1),
		reconcileV1beta1.NewTConfigReconciler(clients, informers, 1),
		reconcileV1beta1.NewTImageReconciler(clients, informers, 1),
		reconcileV1beta1.NewPVCReconciler(clients, informers, 1),
	}

	informers.Start(stopCh)
	if !informers.WaitForCacheSync(stopCh) {
		fmt.Println("WaitForCacheSync Error")
		return
	}

	callbacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			for _, reconcile := range reconciles {
				reconcile.Start(stopCh)
			}
		},
		OnStoppedLeading: func() {
			fmt.Printf("Leaderelection Lost, Program Will Exit\n")
			close(stopCh)
			time.Sleep(time.Second * 5)
			os.Exit(0)
		},
	}

	meta.LeaderElectAndRun(callbacks)
}
