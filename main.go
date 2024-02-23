package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"demo/utils"

	"golang.org/x/sync/singleflight"
)

type Server struct {
	WorkerOpts *Options
	pool       chan int
}

type Options struct {
	GSF *singleflight.Group
}

type Worker struct {
	*Options
}

func (s *Server) NewWorker(i int) *Worker {
	return &Worker{
		Options: s.WorkerOpts,
	}
}

func newWorkerOpts() *Options {
	return &Options{
		GSF: &singleflight.Group{},
	}
}

func NewServer() *Server {
	// 初始化pool
	pool := make(chan int, 200)
	for i := 0; i < 200; i++ {
		pool <- i
	}
	opts := newWorkerOpts()
	return &Server{
		WorkerOpts: opts,
		pool:       pool,
	}
}

func main() {
	server := NewServer()
	// 监听退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	defer func() {
		for i := 0; i < 200; i++ {
			<-server.pool
		}
	}()

	server.Start(quit)
}

func (s *Server) Start(quit chan os.Signal) {
	for {
		select {
		case s := <-quit:
			fmt.Println("退出信号", s)
			return
		case id := <-s.pool:
			s.Poll(id)
		}
	}
}

func (s *Server) Poll(id int) {
	worker := s.NewWorker(id)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				callers := utils.GetCallers(err)
				fmt.Printf("[poller] worker run panic: %q, Call stack:%s\n", err, callers)
			}
			s.pool <- id
		}()

		worker.Do(id)
	}()

	worker.Do(id)
}

func (w *Worker) Do(id int) {
	fmt.Println("worker", id, "doing")

	// 由于业务需要调用外部服务，所以使用singleflight，避免重复调用
	result := w.GSF.DoChan("key", func() (interface{}, error) {
		// mock some external service call
		time.Sleep(2 * time.Second)
		if id == 13 {
			return nil, fmt.Errorf("mock service call error")
		}



		return nil, nil
	})

	res := <-result
	if res.Err != nil {
		fmt.Println("worker", id, "error")
		return
	}

	fmt.Printf("worker %d result: %v\n", id, res.Val)
}
