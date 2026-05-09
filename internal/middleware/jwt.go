package middleware

import (
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/pkg/jwt"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

func StrictAuth(j *jwt.JWT, logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.Request.Header.Get("Authorization")
		if tokenString == "" {
			log.AddNotice(ctx, "token_string", true)
			return
		}
		claims, err := j.ParseToken(tokenString)
		if err != nil {

			api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
			log.Warning(ctx, err)
			ctx.Abort()
			return
		}
		ctx.Set("claims", claims)
		ctx.Set("creator_id", claims.CreatorId)
		ctx.Set("user_id", claims.UserId)
		recoveryLoggerFunc(ctx, logger)
		ctx.Next()
	}
}

func OptionalAuth(j *jwt.JWT, logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.Request.Header.Get("Authorization")
		if tokenString == "" {
			ctx.Next()
			return
		}
		claims, err := j.ParseToken(tokenString)
		if err != nil {
			ctx.Next()
			return
		}
		ctx.Set("claims", claims)
		ctx.Set("creator_id", claims.CreatorId)
		ctx.Set("user_id", claims.UserId)
		recoveryLoggerFunc(ctx, logger)
		ctx.Next()
	}
}

func recoveryLoggerFunc(ctx *gin.Context, logger *log.Logger) {
	// if userInfo, ok := ctx.MustGet("claims").(*jwt.AuthClaims); ok {
	// 	logger.WithValue(ctx, zap.String("UserId", userInfo.UserId))
	// }
}
