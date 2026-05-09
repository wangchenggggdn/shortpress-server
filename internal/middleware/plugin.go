package middleware

import (
	"fmt"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/pkg/crypto"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const (
	PluginIDHeader     = "X-Plugin-Id"
	PluginSecretHeader = "X-Plugin-Secret"
	PluginIDKey        = "plugin_id"
	PluginAuthInfoKey  = "plugin_auth_info"
)

// PluginAuthInfo 插件认证信息
type PluginAuthInfo struct {
	SiteID   string
	PluginID string
	Secret   string
}

// PluginAuthMiddleware 插件认证中间件（只验证 plugin_id 和 secret 的加密签名）
// 职责：只做加密验证，不查询数据库
func PluginAuthMiddleware(v *viper.Viper, logger *log.Logger) gin.HandlerFunc {
	// 在中间件内部创建 crypto 实例
	secretCrypto := crypto.NewPluginSecretCrypto(v)

	return func(ctx *gin.Context) {
		// 1. 从 header 中提取 plugin_id 和 secret
		pluginID := ctx.GetHeader(PluginIDHeader)
		secret := ctx.GetHeader(PluginSecretHeader)

		// 2. 验证必填字段
		if pluginID == "" {
			api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing %s header", PluginIDHeader), nil)
			ctx.Abort()
			return
		}
		if secret == "" {
			api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing %s header", PluginSecretHeader), nil)
			ctx.Abort()
			return
		}

		// 3. 从 context 中获取 site_id（应该由 SiteMiddleware 设置）
		siteID := ctx.GetString(SiteIDKey)
		if siteID == "" {
			api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing site_id in context, ensure SiteMiddleware is applied"), nil)
			ctx.Abort()
			return
		}

		// 4. 使用加密验证密钥（离线验证，不需要查询数据库）
		if !secretCrypto.VerifySecret(pluginID, siteID, secret) {
			logger.WithContext(ctx).Warn(fmt.Sprintf("plugin authentication failed: invalid secret for site_id=%s, plugin_id=%s", siteID, pluginID))
			api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, fmt.Errorf("invalid plugin secret"), nil)
			ctx.Abort()
			return
		}

		// 5. 将认证信息存储在 context 中供后续 handler 使用
		ctx.Set(PluginIDKey, pluginID)
		ctx.Set(PluginAuthInfoKey, &PluginAuthInfo{
			SiteID:   siteID,
			PluginID: pluginID,
			Secret:   secret,
		})

		logger.WithContext(ctx).Info(fmt.Sprintf("plugin authenticated successfully: site_id=%s, plugin_id=%s", siteID, pluginID))

		ctx.Next()
	}
}

// RequirePluginID 是一个只提取 plugin_id 但不验证的中间件
// 用于不需要严格认证但需要 plugin_id 的场景
func RequirePluginID(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		pluginID := ctx.GetHeader(PluginIDHeader)

		if pluginID == "" {
			logger.WithContext(ctx).Warn("missing plugin ID in request header")
		} else {
			ctx.Set(PluginIDKey, pluginID)
		}

		ctx.Next()
	}
}

// GetPluginID 是一个辅助函数，用于从 context 中获取 plugin_id
func GetPluginID(ctx *gin.Context) string {
	if pluginID, exists := ctx.Get(PluginIDKey); exists {
		if id, ok := pluginID.(string); ok {
			return id
		}
	}
	return ""
}

// GetPluginAuthInfo 是一个辅助函数，用于从 context 中获取插件认证信息
func GetPluginAuthInfo(ctx *gin.Context) *PluginAuthInfo {
	if info, exists := ctx.Get(PluginAuthInfoKey); exists {
		if authInfo, ok := info.(*PluginAuthInfo); ok {
			return authInfo
		}
	}
	return nil
}
