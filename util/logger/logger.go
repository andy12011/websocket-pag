//go:generate mockgen -destination=mock_logger/mock_logger.go github.com/andy12011/websocket-pag/sbk/gateway-service/service/gateway/util/logger LoggerInterface

package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Message struct {
	ChainID string `json:"chainID"`
	Level   string `json:"level"`
	Version string `json:"version"`
	Service string `json:"service"`
	Time    string `json:"time"`
	Msg     string `json:"msg"`
}

func InitSysLog(serviceName, level string) (LoggerInterface, error) {
	sysLog := newLogger(getZapLevel(level), serviceName)

	return sysLog, nil
}

func newLogger(level zapcore.Level, serviceName string) *Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(level), // 日志级别
		DisableStacktrace: true,
		Development:       false,         // 开发模式，堆栈跟踪
		Encoding:          "console",     // 输出格式 console 或 json
		EncoderConfig:     encoderConfig, // 编码器配置
		InitialFields: map[string]interface{}{
			"version": "1",
			"service": serviceName,
		},                                    // 初始化字段，如：添加一个服务器名称
		OutputPaths:      []string{"stdout"}, // 输出到指定文件 stdout（标准输出，正常颜色） stderr（错误输出，红色）
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}

	return &Logger{logger: logger}
}

type LoggerInterface interface {
	Debug(ctx context.Context, message string)
	Info(ctx context.Context, message string)
	Warn(ctx context.Context, message string)
	Error(ctx context.Context, err error)
	Panic(ctx context.Context, err error)
}

type Logger struct {
	logger *zap.Logger
}

func (lg *Logger) Debug(ctx context.Context, message string) {
	lg.logger.Debug(message, chainID(ctx))
}

func (lg *Logger) Info(ctx context.Context, message string) {
	lg.logger.Info(message, chainID(ctx))
}

func (lg *Logger) Warn(ctx context.Context, message string) {
	lg.logger.Warn(message, chainID(ctx))
}

func (lg *Logger) Error(ctx context.Context, err error) {
	lg.logger.Error(fmt.Sprintf("%+v", err), chainID(ctx))
}

func (lg *Logger) Panic(ctx context.Context, err error) {
	lg.logger.Panic(fmt.Sprintf("%+v", err), chainID(ctx))
}

func chainID(ctx context.Context) zap.Field {
	chainID := ""
	if s := ctx.Value("ChainID"); s != nil {
		chainID = s.(string)
	}

	return zap.String("chain", chainID)
}

func getZapLevel(l string) zapcore.Level {
	switch l {
	case zapcore.DebugLevel.String(): // "debug"
		return zapcore.DebugLevel
	case zapcore.InfoLevel.String(): // "info"
		return zapcore.InfoLevel
	case zapcore.WarnLevel.String(): // "warn"
		return zapcore.WarnLevel
	case zapcore.ErrorLevel.String(): // "error"
		return zapcore.ErrorLevel
	case zapcore.DPanicLevel.String(): // "dpanic"
		return zapcore.DPanicLevel
	case zapcore.PanicLevel.String(): // "panic"
		return zapcore.PanicLevel
	case zapcore.FatalLevel.String(): // "fatal"
		return zapcore.FatalLevel
	default:
		return zapcore.DebugLevel
	}
}
