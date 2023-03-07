package main

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
	tarsRuntime "k8s.tars.io/runtime"
	"os"
	"tarsagent/gflag"
	"tarsagent/runner"
	"tarsagent/runner/cron"
	"tarsagent/runner/storage"
)

func init() {
	tlvInHost := os.Getenv("TLVInHost")
	if tlvInHost != "" {
		gflag.TLVInHost = tlvInHost
	}

	nodeName := os.Getenv("NodeName")
	if nodeName != "" {
		gflag.NodeName = nodeName
	} else {
		klog.Fatal("env variable NodeName must be set so that this agent can identify itself")
	}
}

func main() {
	runtime.Must(tarsRuntime.CreateContext("", "", false))

	stopCh := make(chan struct{})

	runners := []runner.Runner{
		cron.NewRunner(),
		storage.NewRunner(),
	}

	for _, r := range runners {
		if err := r.Init(); err != nil {
			return
		}
	}

	tarsRuntime.Factories.Start(stopCh)

	for _, r := range runners {
		r.Start(stopCh)
	}

	select {
	case _ = <-stopCh:
		os.Exit(0)
	}
}
