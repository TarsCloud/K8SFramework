package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/tools/leaderelection"
	tarsRuntime "k8s.tars.io/runtime"
	"os"
	"tarscontroller/controller"
	tarsControllerV1beta3 "tarscontroller/controller/tars/v1beta3"
)

func main() {
	stopCh := make(chan struct{})
	err := tarsRuntime.CreateContext("", "")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	controllers := []controller.Controller{
		tarsControllerV1beta3.NewNodeController(1),
		tarsControllerV1beta3.NewDaemonSetController(1),
		tarsControllerV1beta3.NewTTreeController(1),
		tarsControllerV1beta3.NewServiceController(1),
		tarsControllerV1beta3.NewTExitedPodController(1),
		tarsControllerV1beta3.NewStatefulSetController(5),
		tarsControllerV1beta3.NewTServerController(3),
		tarsControllerV1beta3.NewTEndpointController(3),
		tarsControllerV1beta3.NewTAccountController(1),
		tarsControllerV1beta3.NewTConfigController(3),
		tarsControllerV1beta3.NewTImageController(1),
		tarsControllerV1beta3.NewPVCController(1),
	}

	tarsRuntime.Factories.Start(stopCh)

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

	tarsRuntime.LeaderElectAndRun(callbacks, tarsRuntime.Namespace, "tars-controller-manger")
}
