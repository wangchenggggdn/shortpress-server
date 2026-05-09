package middleware

import (
	"shortpress-server/pkg/log"
	"shortpress-server/pkg/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func PrometheusMiddleware(logger *log.Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Collect metrics after request completion
		duration := float64(time.Since(start).Milliseconds())

		// Extract basic info
		method := c.Request.Method
		api := c.Request.URL.Path
		statusCode := strconv.Itoa(c.Writer.Status())

		// Extract business context
		siteID := GetSiteID(c)
		if siteID == "" {
			siteID = "unknown"
		}

		// Record HTTP metrics
		metrics.HTTPRequestsTotal.WithLabelValues(
			method,
			api,
			statusCode,
			siteID,
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			method,
			api,
			siteID,
		).Observe(duration)

	})
}
