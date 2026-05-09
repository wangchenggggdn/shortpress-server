package middleware

import (
	"github.com/gin-gonic/gin"
	"shortpress-server/pkg/log"
)

const (
	languageHeader = "X-Locale" // 请求头名称（不可修改）
	languageKey    = "lang"     // 上下文键名
)

// RequireI18n is a middleware that extracts language from X-Locale header
func RequireI18n(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		lang := ctx.GetHeader(languageHeader)
		// 将语言设置到上下文中
		ctx.Set(languageKey, lang)
		ctx.Next()
	}
}
