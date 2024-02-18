package redis

import (
	"demo/pkg/redis/cache"
	"demo/pkg/redis/option"

	"github.com/google/wire"
)

// ProviderSet .
var ProviderSet = wire.NewSet(
	option.New,
	cache.NewCache,
)
