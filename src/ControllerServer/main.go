package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/tools/leaderelection"
	"os"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"tarscontroller/reconcile/v1beta3"
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
		v1beta3.NewNodeReconciler(clients, informers, 1),
		v1beta3.NewDaemonSetReconciler(clients, informers, 1),
		v1beta3.NewTTreeReconciler(clients, informers, 1),
		v1beta3.NewServiceReconciler(clients, informers, 1),
		v1beta3.NewTExitedPodReconciler(clients, informers, 1),
		v1beta3.NewStatefulSetReconciler(clients, informers, 5),
		v1beta3.NewTServerReconciler(clients, informers, 3),
		v1beta3.NewTEndpointReconciler(clients, informers, 3),
		v1beta3.NewTAccountReconciler(clients, informers, 1),
		v1beta3.NewTConfigReconciler(clients, informers, 3),
		v1beta3.NewTImageReconciler(clients, informers, 1),
		v1beta3.NewPVCReconciler(clients, informers, 1),
		v1beta3.NewTFrameworkConfigReconciler(clients, informers, 1),
	}

	informers.Start(stopCh)
	if !informers.WaitForCacheSync(stopCh) {
		fmt.Println("WaitForCacheSync Error")
		return
	}

	callbacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			for _, reconciler := range reconciles {
				reconciler.Start(stopCh)
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
