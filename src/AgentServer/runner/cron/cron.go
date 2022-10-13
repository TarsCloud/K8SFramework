package cron

import (
	"github.com/robfig/cron/v3"
	"io/ioutil"
	"k8s.io/klog/v2"
	"strings"
)

const (
	CrontabConfigFile = "/etc/tarsagent/crontab.config"
)

type Runner struct {
	crontab *cron.Cron
}

func (r Runner) Init() error {
	return nil
}

func (r Runner) Start(chan struct{}) {
	content, err := ioutil.ReadFile(CrontabConfigFile)
	if err != nil {
		klog.Fatalf("read cron content failed: %s", CrontabConfigFile)
		return
	}
	rules := strings.Split(string(content), "\n")
	r.crontab = cron.New()
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		fields := strings.Split(rule, " ")
		if len(fields) < 7 {
			klog.Errorf("observed unexpected cron: %s, skip", rule)
			continue
		}

		spec := strings.Join(fields[0:6], " ")
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(spec)
		if err != nil {
			klog.Errorf("observed unexpected cron: %s, skip", rule)
			continue
		}
		shell := strings.TrimSpace(strings.Replace(rule, spec, "", 1))
		r.crontab.Schedule(schedule, ShellJob{Shell: shell})
	}
	r.crontab.Start()
	klog.Infof("Ready to start crontab task.")
}

func NewRunner() *Runner {
	return &Runner{}
}
