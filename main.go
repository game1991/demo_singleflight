package main

import (
	"context"
	"demo/worker"
	"flag"
	"fmt"
	"time"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config", "conf/local.toml", "config path")
}

func help() {
	helpInfo := `
    -config	------ config toml file path, default: conf/local.toml
    `
	fmt.Println(helpInfo)
}

func main() {
	flag.Usage = func() {
		help()
	}
	flag.Parse()
	if err := worker.InitConfig(configPath); err != nil {
		panic(err)
	}
	// 获取最近半年用户的域名列表
	uids := "123"
	now := time.Now()
	work, cleanup, err := worker.InitWorker()
	if err != nil {
		panic(err)
	}
	defer cleanup()
	domainList, err := work.GetLastHalfYearUserDomainList(context.Background(), uids, now)
	if err != nil {
		panic(err)
	}

	fmt.Println(domainList)
}
