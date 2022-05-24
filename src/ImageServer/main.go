package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	glPodName      string
	glPodUploadDir string
	glPodBuildDir  string
	glPodCacheDir  string

	glHostBuildDir string
	glHostCacheDir string

	glStopChan chan struct{}

	glK8sContext *K8SContext
	glEngine     *Engine
	glWatcher    *Watcher
	glRestful    *RestfulServer
)

func init() {
	glPodName = os.Getenv("PodName")
	if glPodName == "" {
		log.Printf("get empty PodName value")
		os.Exit(-1)
	}

	workspaceInPod := os.Getenv("WorkSpaceInPod")
	if workspaceInPod == "" {
		log.Printf("get empty WorkSpaceInPod value")
		os.Exit(-1)
	}

	glPodUploadDir = fmt.Sprintf("%s%s/%s", workspaceInPod, UploadDir, glPodName)
	_ = os.RemoveAll(glPodUploadDir)
	if err := os.MkdirAll(glPodUploadDir, 0777); err != nil {
		log.Printf("create upload dir failed: %s", err.Error())
		os.Exit(-1)
	}

	glPodBuildDir = fmt.Sprintf("%s%s/%s", workspaceInPod, BuildDir, glPodName)
	_ = os.RemoveAll(glPodBuildDir)
	if err := os.MkdirAll(glPodBuildDir, 0777); err != nil {
		log.Printf("create build dir failed: %s", err.Error())
		os.Exit(-1)
	}

	glPodCacheDir = fmt.Sprintf("%s%s/%s", workspaceInPod, CacheDir, glPodName)
	_ = os.RemoveAll(glPodCacheDir)
	if err := os.MkdirAll(glPodCacheDir, 0777); err != nil {
		log.Printf("create build dir failed: %s", err.Error())
		os.Exit(-1)
	}

	hostWorkspace := os.Getenv("WorkSpaceInHost")
	if hostWorkspace == "" {
		log.Printf("get empty WorkSpaceInHost value")
		os.Exit(-1)
	}

	glHostBuildDir = fmt.Sprintf("%s%s/%s", hostWorkspace, BuildDir, glPodName)
	glHostCacheDir = fmt.Sprintf("%s%s/%s", hostWorkspace, CacheDir, glPodName)
}

func main() {

	glStopChan = make(chan struct{})

	glK8sContext = CreateK8SContext("", "")

	glWatcher = NewWatcher()
	glWatcher.Start(glStopChan)
	glWatcher.WaitSync(glStopChan)

	glEngine = NewEngine()
	glEngine.Start(glStopChan, MaximumConcurrencyBuildTask)

	glRestful = NewRestful()
	glRestful.Start(glStopChan)

	sigChan := make(chan os.Signal)

	signal.Notify(sigChan, syscall.SIGCHLD, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for true {
		sig := <-sigChan
		switch sig {
		case syscall.SIGCHLD:
			var waitStatus syscall.WaitStatus
			_, _ = syscall.Wait4(-1, &waitStatus, syscall.WNOHANG, nil)
		default:
			break
		}
	}
	close(glStopChan)
	time.Sleep(time.Second * 1)
}
