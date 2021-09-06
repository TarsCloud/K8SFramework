package main

import (
	"github.com/golang/glog"
	"tarsagent/controller"
	"tarsagent/controller/common"
	"tarsagent/crontabtask"
)

func main() {
	// Glog log
	common.InitGlog()

	// Run env
	k8sNamespace, k8sConfig, err := common.LoadEnv()
	if err != nil {
		glog.Errorf("load controller error: %s\n", err)
		return
	}

	// CrontabTask: delete logs and core dumps periodically
	crontabtask.StartCronTabTask()

	// Controller: image downloader and local pv creator/deleter
	controller.StartController(k8sNamespace, k8sConfig)
}
