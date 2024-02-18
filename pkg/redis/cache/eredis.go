package cache

import (
	"context"
	"demo/pkg/redis/option"
	"time"
)

type Cache struct {
	*option.Option
}

func NewCache(opts *option.Option) CacheInterface {
	return &Cache{
		Option: opts,
	}
}

// Get Redis `GET key` command. It returns redis.Nil error when key does not exist.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.MarmotRedis.Client().Get(ctx, key).Result()
}

// Set Redis `SET key value` command.
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.MarmotRedis.Client().Set(ctx, key, value, expiration).Err()
}

// Del Redis `DEL key` command.
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	return c.MarmotRedis.Client().Del(ctx, keys...).Err()
}
