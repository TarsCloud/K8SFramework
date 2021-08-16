package main

import (
	"fmt"
	"tarsagent/controller"
	"tarsagent/controller/common"
	"tarsagent/crontabtask"
)

func main() {
	// run env
	k8sNamespace, k8sConfig, err := common.LoadEnv()
	if err != nil {
		fmt.Printf("load controller error: %s\n", err)
		return
	}

	// Crontab: delete logs and core dumps periodically
	crontabtask.StartCronTabTask()

	// Controller: image downloader and local pv creator/deleter
	controller.StartController(k8sNamespace, k8sConfig)
}
