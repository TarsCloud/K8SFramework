package common

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
)

func InitGlog()  {
	logDir := "/usr/local/app/tars/app_log/tars-agent"

	util := NewVolumeUtil()
	err := util.MakeDir(logDir)
	if err != nil {
		panic(fmt.Errorf("create log dir error: %s\n", err))
	}

	glog.MaxSize = 1024 * 1024 * 100

	_ = flag.Set("log_dir", "/usr/local/app/tars/app_log/tars-agent")
	_ = flag.Set("alsologtostderr", "true")

	flag.Parse()
}
