package middleware

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"shortpress-server/internal/service/cache"
	"shortpress-server/pkg/log"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CacheMiddleware struct {
	cacheService cache.CacheService
	logger       *log.Logger
}

func NewCacheMiddleware(cacheService cache.CacheService, logger *log.Logger) *CacheMiddleware {
	return &CacheMiddleware{
		cacheService: cacheService,
		logger:       logger,
	}
}

// CacheResponse 响应缓存中间件
func (m *CacheMiddleware) CacheResponse(expiration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只缓存GET请求
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// 生成缓存键
		cacheKey := m.generateCacheKey(c)
		m.logger.Debug("Cache key generated", zap.String("key", cacheKey))

		// 尝试获取缓存
		var cachedResponse string
		err := m.cacheService.GetCache(c.Request.Context(), cacheKey, &cachedResponse)
		if err == nil {
			m.logger.WithContext(c.Request.Context()).Debug("Cache hit", zap.String("key", cacheKey))
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", []byte(cachedResponse))
			c.Abort()
			return
		}

		// 包装ResponseWriter以捕获响应
		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = w

		c.Next()

		// 如果响应成功，缓存结果.大小：对于过小的body来说，缓存的必要可能不太大。另外，最重要的也是即便在200的情况下，body 会比较小。可能也会有错。因此不能坐缓存
		MAX_CACHE_BODY_SIZE := 50
		if w.Status() == http.StatusOK && w.body.Len() > MAX_CACHE_BODY_SIZE {
			go func() {
				if err := m.cacheService.SetCache(c.Request.Context(), cacheKey, w.body.String(), expiration); err != nil {
					m.logger.Error("Failed to cache response", zap.Error(err))
				}
			}()
			// c.Header("X-Cache", "MISS")
		}
	}
}

func (m *CacheMiddleware) generateCacheKey(c *gin.Context) string {
	url := c.Request.URL.String()
	siteID := c.GetHeader("X-Site-Id")
	userAgent := c.GetHeader("User-Agent")
	authorization := c.GetHeader("Authorization")
	key := url + siteID + userAgent + authorization
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("response_%x", hash)
}

type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Status() int {
	return w.status
}
