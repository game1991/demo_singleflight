//go:build wireinject
// +build wireinject

package worker

import (
	"demo/pkg/redis"

	"github.com/google/wire"
)

func InitWorker() (*Worker, func(), error) {
	panic(wire.Build(
		wire.Struct(new(Option), "*"),
		NewWorker,
		redis.ProviderSet,
	))
}
