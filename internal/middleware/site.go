package middleware

import (
	"fmt"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

const (
	SiteIDHeader = "X-Site-Id"
	SiteIDKey    = "site_id"
)

// SiteMiddleware extracts the SiteID from request headers and sets it in the context
func SiteMiddleware(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		siteID := ctx.GetHeader(SiteIDHeader)
		if siteID == "" {
			api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing %s header", SiteIDHeader), nil)
			ctx.Abort()
			return
		}

		// Set the site ID in the context for downstream handlers
		ctx.Set(SiteIDKey, siteID)
		ctx.Next()
	}
}

// RequireSiteID is a middleware that requires the X-Site-Id header to be present
// but doesn't abort if it's missing - simply logs a warning
func RequireSiteID(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		siteID := ctx.GetHeader(SiteIDHeader)

		if siteID == "" {
			logger.WithContext(ctx).Warn("missing site ID in request header")
		} else {
			ctx.Set(SiteIDKey, siteID)
		}

		ctx.Next()
	}
}

// GetSiteID is a helper function to retrieve the site ID from the context
func GetSiteID(ctx *gin.Context) string {
	if siteID, exists := ctx.Get(SiteIDKey); exists {
		if id, ok := siteID.(string); ok {
			return id
		}
	}
	return ""
}
