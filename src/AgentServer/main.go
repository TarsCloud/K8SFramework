package main

import (
	"k8s.io/klog/v2"
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

	for _, r := range runners {
		r.Start(stopCh)
	}

	select {
	case _ = <-stopCh:
		os.Exit(0)
	}
}
