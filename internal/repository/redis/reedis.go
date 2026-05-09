package redis

import (
	"context"
	"encoding/json"
	"time"

	"shortpress-server/pkg/log"

	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	rdb    *redis.Client
	logger *log.Logger
}

func NewRedisRepository(logger *log.Logger, rdb *redis.Client) *RedisRepository {
	return &RedisRepository{
		rdb:    rdb,
		logger: logger,
	}
}

// 通用 Redis 操作接口
type BaseRedisOperation interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// 实现基础操作
func (r *RedisRepository) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, key, data, expiration).Err()
}

func (r *RedisRepository) Get(ctx context.Context, key string, dest interface{}) error {
	result, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(result), dest)
}

func (r *RedisRepository) Del(ctx context.Context, keys ...string) error {
	return r.rdb.Del(ctx, keys...).Err()
}

func (r *RedisRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.rdb.Exists(ctx, key).Result()
	return count > 0, err
}

func (r *RedisRepository) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return r.rdb.SetNX(ctx, key, data, expiration).Result()
}

func (r *RedisRepository) Incr(ctx context.Context, key string) (int64, error) {
	return r.rdb.Incr(ctx, key).Result()
}

func (r *RedisRepository) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.rdb.Expire(ctx, key, expiration).Err()
}
