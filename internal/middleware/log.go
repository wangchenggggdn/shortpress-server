package middleware

import (
	"bytes"
	"fmt"
	"strings"
	"shortpress-server/pkg/log"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// maxResponseBodyLogLen 单条响应体写入日志的最大字符数，避免超大 JSON/文件接口刷爆日志。
const maxResponseBodyLogLen = 8192

func RequestLogMiddleware(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startTime := time.Now()
		requestID := uuid.New().String()

		// 初始化请求的业务字段容器
		log.InitNoticeFields(ctx)

		// 设置请求ID到上下文
		logger.WithValue(ctx, zap.String("request_id", requestID))
		logger.WithValue(ctx, zap.String("api", ctx.Request.URL.Path))
		logger.WithValue(ctx, zap.String("query_param", ctx.Request.URL.RawQuery))

		ctx.Next()

		// 在请求结束时记录完整信息
		duration := time.Since(startTime)

		// 从上下文获取业务信息
		creatorID := ctx.GetString("creator_id")
		userID := ctx.GetString("user_id")
		siteID := ctx.GetString("site_id")

		// 构建基础字段
		fields := []zap.Field{
			zap.String("method", ctx.Request.Method),
			zap.Int("status_code", ctx.Writer.Status()),
			zap.String("user_agent", ctx.Request.UserAgent()),
			zap.Int64("duration_ms", duration.Milliseconds()),
		}

		// 添加业务上下文
		if creatorID != "" {
			fields = append(fields, zap.String("creator_id", creatorID))
		}
		if userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}
		if siteID != "" {
			fields = append(fields, zap.String("site_id", siteID))
		}

		// 获取业务字段并添加到日志中
		noticeFields := log.GetNoticeFields(ctx)
		businessFields := noticeFields.ToZapFields()
		fields = append(fields, businessFields...)

		// 输出 notice 类型的结构化日志
		logger.WithContext(ctx).Info(string(log.LogTypeNotice), fields...)
	}
}
func ResponseLogMiddleware(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		if !strings.HasPrefix(path, "/api") {
			ctx.Next()
			return
		}

		blw := &bodyLogWriter{body: bytes.NewBuffer(nil), ResponseWriter: ctx.Writer}
		ctx.Writer = blw
		ctx.Next()

		raw := blw.body.Bytes()
		payload := summarizeResponseBodyForLog(raw)
		logger.WithContext(ctx).Info(string(log.LogTypeNotice),
			zap.String("response_log", "body"),
			zap.Int("status_code", ctx.Writer.Status()),
			zap.String("response_body", payload),
		)
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	_, _ = w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	_, _ = w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func summarizeResponseBodyForLog(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	if !utf8.Valid(raw) {
		return fmt.Sprintf("<non-utf8 binary, %d bytes>", len(raw))
	}
	if len(raw) <= maxResponseBodyLogLen {
		return string(raw)
	}
	cut := raw[:maxResponseBodyLogLen]
	for len(cut) > 0 && !utf8.Valid(cut) {
		cut = cut[:len(cut)-1]
	}
	return string(cut) + fmt.Sprintf("...(truncated, total %d bytes)", len(raw))
}
