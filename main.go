package main

import (
	"context"
	"demo/worker"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/gotomicro/ego/core/elog"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(200)
	for i := 0; i < 200; i++ {
		go func(i int) {
			defer wg.Done()
			domainList, err := work.GetLastHalfYearUserDomainList(ctx, uids, now)
			if err != nil {
				elog.Error("GetLastHalfYearUserDomainList", elog.Any("协程号i", i), elog.FieldErr(err))
				return
			}
			fmt.Printf("协程%d执行完,res=%v", i, domainList)
		}(i)
	}
	wg.Wait()
}
