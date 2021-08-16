package crontabtask

import (
	"bytes"
	"github.com/golang/glog"
	"github.com/robfig/cron/v3"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"
)

const (
	CrontabConfigFile = "/etc/tarsagent/crontab.config"
)

func StartCronTabTask() {
	crontabContent ,err := ioutil.ReadFile(CrontabConfigFile)
	if err != nil {
		glog.Errorf("error to readFile: %s\n", CrontabConfigFile)
		return
	}
	crontabRules := strings.Split(string(crontabContent), "\n")

	crontab := cron.New()
	for _, rule := range crontabRules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		fields := strings.Split(rule, " ")

		// Only support standard format with 6 fields now
		sched := strings.Join(fields[:6], " ")
		shell := strings.TrimSpace(strings.Replace(rule, sched, "", 1))

		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(sched)
		if err != nil {
			glog.Errorf("error to addFunc, sched: %s, shell: \"%s\", msg: %s\n", sched, shell, err.Error())
		}

		_ = crontab.Schedule(schedule, NewShellJob(shell))
	}
	crontab.Start()
	glog.Infof("Ready to start crontab task.")
}

func NewShellJob(shell string) ShellJob {
	return ShellJob{Shell: shell}
}

type ShellJob struct {
	Shell string
}

func (job ShellJob) Run() {
	cmd := exec.Command("/bin/sh", "-c", job.Shell)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		glog.Errorf("%s: error to execute \"%s\", msg: %s\n", time.Now().String(), job.Shell, err.Error())
	} else {
		glog.Infof("%s: succ. to execute \"%s\", msg: %s\n", time.Now().String(), job.Shell, out.String())
	}
}
