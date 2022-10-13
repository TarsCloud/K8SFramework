package main

import (
	"tarsagent/runner"
	"tarsagent/runner/cron"
	"tarsagent/runner/storage"
	"time"
)

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

	for {
		time.Sleep(time.Second * 3)
	}
}
