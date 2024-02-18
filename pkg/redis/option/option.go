package option

import (
	"github.com/gotomicro/ego-component/eredis"
)

type Option struct {
	MarmotRedis *eredis.Component
}

func New() *Option {
	return &Option{
		MarmotRedis: eredis.Load("redis.marmot").Build(),
	}
}