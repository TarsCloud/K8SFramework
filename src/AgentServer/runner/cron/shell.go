package cron

import (
	"bytes"
	"k8s.io/klog/v2"
	"os/exec"
)

type ShellJob struct {
	Shell string
}

func (job ShellJob) Run() {
	cmd := exec.Command("/bin/sh", "-c", job.Shell)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		klog.Errorf("execute command \"%s\" failed: %s", job.Shell, err.Error())
	} else {
		klog.Infof("execute command \"%s\" success, and get output:\n%s", job.Shell, out.String())
	}
}
