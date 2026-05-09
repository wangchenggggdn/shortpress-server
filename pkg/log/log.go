package log

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogType 定义日志类型
type LogType string

const (
	LogTypeRequest       LogType = "REQUEST_START"
	LogTypeResponse      LogType = "REQUEST_END"
	LogTypeNotice        LogType = "NOTICE"
	LogTypeWarning       LogType = "WARNING"
	LogTypeError         LogType = "ERROR"
	LogTypeDataOperation LogType = "DATA_OPERATION"
)

// NoticeFields 业务字段映射，线程安全
type NoticeFields struct {
	mu     sync.RWMutex
	fields map[string]interface{}
}

// NewNoticeFields 创建新的业务字段容器
func NewNoticeFields() *NoticeFields {
	return &NoticeFields{
		fields: make(map[string]interface{}),
	}
}

// Set 设置字段值（线程安全）
func (nf *NoticeFields) Set(key string, value interface{}) {
	nf.mu.Lock()
	defer nf.mu.Unlock()
	nf.fields[key] = value
}

// Get 获取字段值（线程安全）
func (nf *NoticeFields) Get(key string) (interface{}, bool) {
	nf.mu.RLock()
	defer nf.mu.RUnlock()
	value, exists := nf.fields[key]
	return value, exists
}

// GetAll 获取所有字段（线程安全）
func (nf *NoticeFields) GetAll() map[string]interface{} {
	nf.mu.RLock()
	defer nf.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range nf.fields {
		result[k] = v
	}
	return result
}

// ToZapFields 转换为 zap.Field 数组
func (nf *NoticeFields) ToZapFields() []zap.Field {
	nf.mu.RLock()
	defer nf.mu.RUnlock()

	fields := make([]zap.Field, 0, len(nf.fields))
	for key, value := range nf.fields {
		switch v := value.(type) {
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case int64:
			fields = append(fields, zap.Int64(key, v))
		case float64:
			fields = append(fields, zap.Float64(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))
		case time.Duration:
			fields = append(fields, zap.Duration(key, v))
		default:
			fields = append(fields, zap.Any(key, v))
		}
	}
	return fields
}

// 定义 context 键
type loggerKey struct{}
type noticeFieldsKey struct{}

var ctxLoggerKey = loggerKey{}
var ctxNoticeFieldsKey = noticeFieldsKey{}

type Logger struct {
	*zap.Logger
}

func NewLog(conf *viper.Viper) *Logger {
	// log address "out.log" User-defined
	lp := conf.GetString("log.log_file_name")
	lv := conf.GetString("log.log_level")
	var level zapcore.Level
	//debug<info<warn<error<fatal<panic
	switch lv {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}
	hook := lumberjack.Logger{
		Filename:   lp,                             // Log file path
		MaxSize:    conf.GetInt("log.max_size"),    // Maximum size unit for each log file: M
		MaxBackups: conf.GetInt("log.max_backups"), // The maximum number of backups that can be saved for log files
		MaxAge:     conf.GetInt("log.max_age"),     // Maximum number of days the file can be saved
		Compress:   conf.GetBool("log.compress"),   // Compression or not
	}

	var encoder zapcore.Encoder
	if conf.GetString("log.encoding") == "console" {
		// Console for all levels
		encoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "Logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.FullCallerEncoder,
		})
	} else {
		// JSON for non-error, Console for error and above to make stack traces multi-line
		jsonEncoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
		consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "Logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.FullCallerEncoder,
		})

		ws := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&hook))
		jsonCore := zapcore.NewCore(jsonEncoder, ws, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			// honor configured minimum level and only log below Error here
			return lvl < zap.ErrorLevel && lvl >= level
		}))
		consoleCore := zapcore.NewCore(consoleEncoder, ws, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			// error and above go to console encoder for better readability
			return lvl >= zap.ErrorLevel && lvl >= level
		}))
		core := zapcore.NewTee(jsonCore, consoleCore)
		if conf.GetString("env") != "prod" {
			return &Logger{zap.New(core, zap.Development(), zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}
		}
		return &Logger{zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}
	}
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&hook)),
		level,
	)
	if conf.GetString("env") != "prod" {
		return &Logger{zap.New(core, zap.Development(), zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}
	}
	return &Logger{zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}
}

// WithValue Adds a field to the specified context
func (l *Logger) WithValue(ctx context.Context, fields ...zapcore.Field) context.Context {
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
		c.Request = c.Request.WithContext(context.WithValue(ctx, ctxLoggerKey, l.WithContext(ctx).With(fields...)))
		return c
	}
	return context.WithValue(ctx, ctxLoggerKey, l.WithContext(ctx).With(fields...))
}

// WithContext Returns a zap instance from the specified context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
	}
	zl := ctx.Value(ctxLoggerKey)
	ctxLogger, ok := zl.(*zap.Logger)
	if ok {
		return &Logger{ctxLogger}
	}
	return l
}

// InitNoticeFields 初始化请求的业务字段容器到上下文中
func InitNoticeFields(ctx context.Context) context.Context {
	nf := NewNoticeFields()
	if c, ok := ctx.(*gin.Context); ok {
		c.Set("notice_fields", nf)
		reqCtx := context.WithValue(c.Request.Context(), ctxNoticeFieldsKey, nf)
		c.Request = c.Request.WithContext(reqCtx)
		return c
	}
	return context.WithValue(ctx, ctxNoticeFieldsKey, nf)
}

// GetNoticeFields 从上下文获取业务字段容器
func GetNoticeFields(ctx *gin.Context) *NoticeFields {
	if nf, exists := ctx.Get("notice_fields"); exists {
		if noticeFields, ok := nf.(*NoticeFields); ok {
			return noticeFields
		}
	}
	// ctx = ctx.Request.Context()

	if nf := ctx.Value(ctxNoticeFieldsKey); nf != nil {
		if noticeFields, ok := nf.(*NoticeFields); ok {
			return noticeFields
		}
	}

	// 如果没有找到，返回一个新的实例
	return NewNoticeFields()
}

// AddNotice 添加业务字段到当前请求上下文（线程安全）
func AddNotice(ctx *gin.Context, key string, value interface{}) {
	noticeFields := GetNoticeFields(ctx)
	noticeFields.Set(key, value)
}

func Info(ctx *gin.Context, value interface{}) {
	// 从上下文获取 logger
	zl := ctx.Value(ctxLoggerKey)
	ctxLogger, ok := zl.(*zap.Logger)
	if !ok {
		// 如果没有从上下文获取到 logger，使用默认方式
		logger := &Logger{zap.L()}
		ctxLogger = logger.WithContext(ctx).Logger
	}
	key := "content"
	ctxLogger = ctxLogger.WithOptions(zap.AddCallerSkip(1))

	// 直接输出信息日志
	switch v := value.(type) {
	case string:
		ctxLogger.Info(string(LogTypeNotice), zap.String(key, v))
	case error:
		ctxLogger.Info(string(LogTypeNotice), zap.String(key, v.Error()))
	case int:
		ctxLogger.Info(string(LogTypeNotice), zap.Int(key, v))
	case int64:
		ctxLogger.Info(string(LogTypeNotice), zap.Int64(key, v))
	case float64:
		ctxLogger.Info(string(LogTypeNotice), zap.Float64(key, v))
	case bool:
		ctxLogger.Info(string(LogTypeNotice), zap.Bool(key, v))
	default:
		ctxLogger.Info(string(LogTypeNotice), zap.Any(key, v))
	}
}

func Warning(ctx *gin.Context, value interface{}) {
	// 从上下文获取 logger
	zl := ctx.Value(ctxLoggerKey)
	ctxLogger, ok := zl.(*zap.Logger)
	if !ok {
		// 如果没有从上下文获取到 logger，使用默认方式
		logger := &Logger{zap.L()}
		ctxLogger = logger.WithContext(ctx).Logger
	}
	key := "content"
	ctxLogger = ctxLogger.WithOptions(zap.AddCallerSkip(1))

	// 直接输出警告日志
	switch v := value.(type) {
	case string:
		ctxLogger.Warn(string(LogTypeWarning), zap.String(key, v))
	case error:
		ctxLogger.Warn(string(LogTypeWarning), zap.String(key, v.Error()))
	case int:
		ctxLogger.Warn(string(LogTypeWarning), zap.Int(key, v))
	case int64:
		ctxLogger.Warn(string(LogTypeWarning), zap.Int64(key, v))
	case float64:
		ctxLogger.Warn(string(LogTypeWarning), zap.Float64(key, v))
	case bool:
		ctxLogger.Warn(string(LogTypeWarning), zap.Bool(key, v))
	default:
		ctxLogger.Warn(string(LogTypeWarning), zap.Any(key, v))
	}
}

func Error(ctx *gin.Context, value interface{}) {
	// 从上下文获取 logger
	zl := ctx.Value(ctxLoggerKey)
	ctxLogger, ok := zl.(*zap.Logger)
	if !ok {
		// 如果没有从上下文获取到 logger，使用默认方式
		logger := &Logger{zap.L()}
		ctxLogger = logger.WithContext(ctx).Logger
	}

	key := "content"
	ctxLogger = ctxLogger.WithOptions(zap.AddCallerSkip(1))
	switch v := value.(type) {
	case string:
		ctxLogger.Error(string(LogTypeError), zap.String(key, v))
	case error:
		ctxLogger.Error(string(LogTypeError), zap.String(key, v.Error()))
	case int:
		ctxLogger.Error(string(LogTypeError), zap.Int(key, v))
	case int64:
		ctxLogger.Error(string(LogTypeError), zap.Int64(key, v))
	case float64:
		ctxLogger.Error(string(LogTypeError), zap.Float64(key, v))
	case bool:
		ctxLogger.Error(string(LogTypeError), zap.Bool(key, v))
	default:
		ctxLogger.Error(string(LogTypeError), zap.Any(key, v))
	}
}
