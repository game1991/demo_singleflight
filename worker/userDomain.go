package worker

import (
	"context"
	"demo/pkg/redis/cache"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/core/util/xstring"
	"golang.org/x/sync/singleflight"
)

type Option struct {
	Cache cache.CacheInterface
	GSF   singleflight.Group
}

type Worker struct {
	*Option
}

func NewWorker(opts *Option) *Worker {
	return &Worker{
		Option: opts,
	}
}

// 获取最近半年用户的域名列表
func (w *Worker) GetLastHalfYearUserDomainList(ctx context.Context, uids string, now time.Time) (domainList []string, err error) {
	var (
		step = "success"
		key  string
	)
	defer func() {
		logFields := []elog.Field{
			elog.FieldCtxTid(ctx),
			elog.String("step", step),
			elog.String("uids", uids),
			elog.String("now", fmt.Sprintf("%d", now.Unix())),
			elog.String("key", key),
			elog.Any("domainList", xstring.JSON(domainList)),
		}
		if err != nil {
			logFields = append(logFields, elog.Any("err", err))
			elog.Error("GetLastHalfYearUserDomainList", logFields...)
		} else {
			elog.Info("GetLastHalfYearUserDomainList", logFields...)
		}
	}()
	// 1秒内的数据，使用同一秒时间戳作为key：uids+_+timestamp
	key = PerfixUserDomain + uids + "_" + fmt.Sprintf("%d", now.Unix())

	// 读取缓存
	res, err := w.Cache.Get(ctx, key)
	if err == nil {
		elog.Debug("Cache.Get", elog.String("res", xstring.PrettyJSON(res)))
		if err = json.Unmarshal([]byte(res), &domainList); err != nil {
			step = "json.Unmarshal failed"
			return
		}
		return
	}
	if err != redis.Nil && !errors.Is(err, redis.Nil) {
		step = "cache.Get failed"
		return
	}

	// 缓存不存在，从接口读取
	result := w.GSF.DoChan(key, func() (interface{}, error) {
		domainList, err := w.getUserDomainList(ctx, uids, now)
		if err != nil {
			step = "getUserDomainList failed"
			return nil, err
		}
		// 设置缓存
		bts, err := json.Marshal(domainList)
		if err != nil {
			step = "json.Marshal domainList failed"
			return nil, err
		}
		if err = w.Cache.Set(ctx, key, string(bts), time.Second*5); err != nil {
			step = "cache.Set failed"
			return nil, err
		}
		return domainList, nil
	})
	select {
	case v := <-result:
		if v.Err != nil {
			step = "w.GSF.DoChan failed"
			return nil, v.Err
		}
		return v.Val.([]string), nil
	case <-ctx.Done():
		step = "w.GSF.DoChan timeout"
		return nil, ctx.Err()
	}
}

var (
	count int
	mutex sync.Mutex
)

func (w *Worker) getUserDomainList(ctx context.Context, uids string, now time.Time) (domainList []string, err error) {
	mutex.Lock()
	count++
	mutex.Unlock()
	elog.Info("getUserDomainList 被调用次数：", elog.Int("count", count))
	return []string{}, nil
}
