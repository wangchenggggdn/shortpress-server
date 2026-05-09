package cache

import (
	"context"
	"fmt"
	"time"

	"shortpress-server/internal/repository/redis"
	"shortpress-server/pkg/log"
)

type CacheService struct {
	redis  *redis.RedisRepository
	logger *log.Logger
}

func NewCacheService(redis *redis.RedisRepository, logger *log.Logger) *CacheService {
	return &CacheService{
		redis:  redis,
		logger: logger,
	}
}

// BuildKey generates a cache key
func (s *CacheService) BuildKey(prefix, id string) string {
	return fmt.Sprintf("%s:%s", prefix, id)
}

// 通用缓存获取方法
func (s *CacheService) GetCache(ctx context.Context, key string, dest interface{}) error {
	return s.redis.Get(ctx, key, dest)
}

// 通用缓存设置方法
func (s *CacheService) SetCache(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return s.redis.Set(ctx, key, value, expiration)
}

// 删除缓存
func (s *CacheService) DelCache(ctx context.Context, keys ...string) error {
	return s.redis.Del(ctx, keys...)
}

// 检查缓存是否存在
func (s *CacheService) ExistsCache(ctx context.Context, key string) (bool, error) {
	return s.redis.Exists(ctx, key)
}

// 设置分布式锁
func (s *CacheService) SetLock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return s.redis.SetNX(ctx, key, "locked", expiration)
}

// 计数器递增
func (s *CacheService) IncrCounter(ctx context.Context, key string) (int64, error) {
	return s.redis.Incr(ctx, key)
}
