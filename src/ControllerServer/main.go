package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/tools/leaderelection"
	"os"
	"tarscontroller/controller"
	"tarscontroller/controller/v1beta3"
	"tarscontroller/util"
	"tarscontroller/webhook"
	"time"
)

func main() {

	stopCh := make(chan struct{})

	clients, factories, err := util.CreateContext("", "")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	//call webhook.start() before factories.start(),because factories.start() depend on webhook.conversion service
	hooks := webhook.New(clients, factories)
	hooks.Start(stopCh)

	time.Sleep(time.Second * 1)

	controllers := []controller.Controller{
		v1beta3.NewNodeController(clients, factories, 1),
		v1beta3.NewDaemonSetController(clients, factories, 1),
		v1beta3.NewTTreeController(clients, factories, 1),
		v1beta3.NewServiceController(clients, factories, 1),
		v1beta3.NewTExitedPodController(clients, factories, 1),
		v1beta3.NewStatefulSetController(clients, factories, 5),
		v1beta3.NewTServerController(clients, factories, 3),
		v1beta3.NewTEndpointController(clients, factories, 3),
		v1beta3.NewTAccountController(clients, factories, 1),
		v1beta3.NewTConfigController(clients, factories, 3),
		v1beta3.NewTImageController(clients, factories, 1),
		v1beta3.NewPVCController(clients, factories, 1),
	}

	factories.Start(stopCh)

	callbacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			for _, c := range controllers {
				go c.Run(stopCh)
			}
		},
		OnStoppedLeading: func() {
			fmt.Printf("Leaderelection Lost, Program Will Exit\n")
			close(stopCh)
			os.Exit(0)
		},
	}

	util.LeaderElectAndRun(callbacks)
}
