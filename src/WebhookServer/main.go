package main

import (
	"fmt"
	tarsRuntime "k8s.tars.io/runtime"
	"tarswebhook/webhook"
)

func main() {
	stopCh := make(chan struct{})
	err := tarsRuntime.CreateContext("", "", false)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	hooks := webhook.New()
	tarsRuntime.Factories.Start(stopCh)
	hooks.Run(stopCh)
}
