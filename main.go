package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"

	"log"

	"demo/utils"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/sync/singleflight"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// 设置日志输出到本地文件
	log.SetOutput(openLogFile())
}

func openLogFile() *os.File {
	f, err := os.OpenFile("demo.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	return f
}

type Server struct {
	WorkerOpts *Options
	pool       chan int
}

type Options struct {
	Cache    *redis.Client
	LRUCache *lru.Cache
	GSF      *singleflight.Group
}

type Worker struct {
	*Options
}

func (s *Server) NewWorker(i int) *Worker {
	return &Worker{
		Options: s.WorkerOpts,
	}
}

type LoggerHook struct {
	logger *log.Logger
}

func GetCaller(depth int, trace bool) string {
	callers := ""
	for i := 0; true; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		if trace {
			callers = callers + fmt.Sprintf("%v:%v\n", file, line)
		} else {
			callers = fmt.Sprintf("%v:%v\n", file, line)
		}

		if i == depth {
			break
		}
	}
	return callers
}

// 请将LoogerHook实现redis.Hook
func (l *LoggerHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (l *LoggerHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	l.logger.Printf("%s cmd: %v\n", GetCaller(5, false), cmd)
	return nil
}

func (l *LoggerHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (l *LoggerHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	l.logger.Printf("%s cmds: %v\n", GetCaller(5, false), cmds)
	return nil
}

func NewReidsClient() *redis.Client {
	// 初始化redis
	reidsCli := redis.NewClient(&redis.Options{
		// 设置 Redis 服务器的地址
		Addr: "127.0.0.1:6379",
		// 设置密码
		Password: "foobared",
		// 将要使用的数据库设置为默认数据库
		DB: 0,

		// 设置最大重试次数
		MaxRetries: 3,
		// 设置在重试失败操作之前等待的最短时间
		MinRetryBackoff: 8 * time.Millisecond,
		// 设置在重试失败操作之前等待的最长时间
		MaxRetryBackoff: 512 * time.Millisecond,

		// 设置等待建立连接的最长时间
		DialTimeout: 5 * time.Second,
		// 设置等待读操作完成的最长时间
		ReadTimeout: 3 * time.Second,
		// 设置等待写操作完成的最长时间
		WriteTimeout: 3 * time.Second,

		// 设置连接池中的最大连接数
		PoolSize: 10,
		// 设置从连接池获取连接的最长等待时间
		PoolTimeout: 30 * time.Second,
		// 设置连接池中的最小空闲连接数
		MinIdleConns: 10,

		// 设置连接在连接池中的最长空闲时间
		IdleTimeout: 60 * time.Second,

		// 设置连接的最长生命周期
		MaxConnAge: 20 * time.Minute,
	})
	// 初始化logger
	logger := log.New(openLogFile(), "", log.LstdFlags)

	reidsCli.AddHook(&LoggerHook{logger: logger})
	return reidsCli
}

func newWorkerOpts() *Options {
	redisCli := NewReidsClient()
	lruCache, err := lru.New(100)
	if err != nil {
		panic(err)
	}
	return &Options{
		Cache:    redisCli,
		LRUCache: lruCache,
		GSF:      &singleflight.Group{},
	}
}

func NewServer() *Server {
	// 初始化pool
	pool := make(chan int, 200)
	for i := 0; i < 10; i++ {
		pool <- i + 1
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
	fmt.Println("服务启动中...")
	server.Start(quit)
}

func (s *Server) Start(quit chan os.Signal) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case s := <-quit:
			fmt.Println("退出信号", s)
			return
		case id := <-s.pool:
			s.Poll(ctx, id)
		}
	}
}

func (s *Server) Poll(ctx context.Context, id int) {
	worker := s.NewWorker(id)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				callers := utils.GetCallers(err)
				log.Printf("[poller] worker run panic: %q, Call stack:%s\n", err, callers)
			}
			s.pool <- id
		}()
		key := strconv.Itoa(int(time.Now().Unix())) // 以当前时刻作为key，精度到秒
		worker.Do(ctx, id, key)
	}()
}

func (w *Worker) Do(ctx context.Context, id int, key string) {
	log.Printf("worker[%d] doning...[key]=%s\n", id, key)
	// 先从本地lru缓存中读取
	data, ok := w.LRUCache.Get(key)
	if ok {
		log.Printf("worker[%d] get lru cache result: %v\n", id, data)
		return
	}

	data, err := w.Cache.Get(ctx, key).Result()
	if err != nil && err != redis.Nil && !errors.Is(err, redis.Nil) {
		panic("获取缓存失败" + err.Error())
	}
	if err == nil {
		log.Printf("worker[%d] get cache result: %v\n", id, data)
		return
	}

	// 由于业务需要调用外部服务，所以使用singleflight，避免重复调用
	result := w.GSF.DoChan(key, func() (interface{}, error) {
		// mock some external service call
		time.Sleep(2 * time.Second)
		if id == 13 {
			return nil, fmt.Errorf("mock service call error")
		}
		data := "data" + uuid.New().String()

		// 存入本地缓存
		w.LRUCache.Add(key, data)

		_, err := w.Cache.SetEX(ctx, key, data, time.Minute).Result()
		if err != nil {
			return nil, fmt.Errorf("set cache error: %w", err)
		}

		return data, nil
	})

	res := <-result
	if res.Err != nil {
		log.Println("worker", id, "error=", res.Err)
		return
	}

	log.Printf("worker %d result: %v\n", id, res.Val)
}
