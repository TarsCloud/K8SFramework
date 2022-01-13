package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/tools/leaderelection"
	"os"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	reconcileV1beta2 "tarscontroller/reconcile/v1beta2"
	"tarscontroller/webhook"
	"time"
)

func main() {

	stopCh := make(chan struct{})

	clients, informers, err := controller.CreateContext("", "")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	hooks := webhook.New(clients, informers)
	//call webhook.start() before informers.start(),because informers.start() depend on conversion webhook service

	hooks.Start(stopCh)
	time.Sleep(time.Second * 1)

	// new reconcile should before call informers.start() => because reconcile should registry into informers
	reconciles := []reconcile.Reconcile{
		reconcileV1beta2.NewNodeReconciler(clients, informers, 1),
		reconcileV1beta2.NewTDeployReconciler(clients, informers, 1),
		reconcileV1beta2.NewDaemonSetReconciler(clients, informers, 1),
		reconcileV1beta2.NewTTreeReconciler(clients, informers, 1),
		reconcileV1beta2.NewServiceReconciler(clients, informers, 1),
		reconcileV1beta2.NewTExitedPodReconciler(clients, informers, 1),
		reconcileV1beta2.NewStatefulSetReconciler(clients, informers, 3),
		reconcileV1beta2.NewTServerReconciler(clients, informers, 3),
		reconcileV1beta2.NewTEndpointReconciler(clients, informers, 3),
		reconcileV1beta2.NewTAccountReconciler(clients, informers, 1),
		reconcileV1beta2.NewTConfigReconciler(clients, informers, 1),
		reconcileV1beta2.NewTImageReconciler(clients, informers, 1),
		reconcileV1beta2.NewPVCReconciler(clients, informers, 1),
		reconcileV1beta2.NewTFrameworkConfigReconciler(clients, informers, 1),
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
			os.Exit(0)
		},
	}

	controller.LeaderElectAndRun(callbacks)
}
