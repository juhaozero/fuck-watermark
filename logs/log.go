package logs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	defaultLogger *Logger
	once          sync.Once
	initErr       error
)

// Logger 对 zap.Logger 的封装，支持日志切分与结构化输出。
type Logger struct {
	zap   *zap.Logger
	sugar *zap.SugaredLogger
	file  *lumberjack.Logger
}

// Init 使用配置初始化全局日志实例，多次调用仅第一次生效。
func Init(cfg Config) error {
	once.Do(func() {
		defaultLogger, initErr = New(cfg)
	})
	return initErr
}

// New 根据配置创建日志实例。
func New(cfg Config) (*Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	encoder := newEncoder(cfg.Format)
	writers := make([]zapcore.WriteSyncer, 0, 2)
	var fileWriter *lumberjack.Logger

	if cfg.Filename != "" && cfg.LogPath != "" {
		fileWriter = &lumberjack.Logger{
			Filename:   filepath.Join(cfg.LogPath, fmt.Sprintf("%s.log", cfg.Filename)),
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups, // 保留个数
			MaxAge:     cfg.MaxAge,     // 保留天数
			Compress:   cfg.Compress,
			LocalTime:  true,
		}
		writers = append(writers, zapcore.AddSync(fileWriter))
	}

	// 如果配置了文件路径，则输出到文件，否则输出到控制台
	if cfg.Console || cfg.Filename == "" || cfg.LogPath == "" {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}
	if cfg.IsDebug {
		level = zap.DebugLevel
	}
	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writers...), level)
	// Skip(2)：包级 Infof 等 → Logger.Infof → sugar；直接 Default().Infof 时 caller 会偏一层
	z := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2), zap.AddStacktrace(zapcore.ErrorLevel))

	// 调试模式
	if cfg.IsDebug {
		z = z.WithOptions(zap.Development())
	}
	return &Logger{
		zap:   z,
		sugar: z.Sugar(),
		file:  fileWriter,
	}, nil
}

// Default 返回全局日志实例，未初始化时返回 no-op 日志。
func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger, _ = New(Config{
			Level:   "info",
			Format:  "console",
			Console: true,
		})
	}
	return defaultLogger
}

// Writer 返回供使用的 io.Writer
func (l *Logger) Writer() io.Writer {
	if l.file != nil {
		return l.file
	}
	return os.Stdout
}

// Zap 返回底层 zap.Logger，便于与第三方库集成。
func (l *Logger) Zap() *zap.Logger {
	return l.zap
}

// Sugar 返回 SugaredLogger。
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.sugar
}

// With 追加结构化字段，返回新的 Logger 实例。
func (l *Logger) With(fields ...zap.Field) *Logger {
	z := l.zap.With(fields...)
	return &Logger{zap: z, sugar: z.Sugar(), file: l.file}
}

// Named 为日志添加子模块名称。
func (l *Logger) Named(name string) *Logger {
	z := l.zap.Named(name)
	return &Logger{zap: z, sugar: z.Sugar(), file: l.file}
}

// Sync 刷盘缓冲日志。
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Close 刷盘并关闭日志文件，适用于进程退出前调用。
func (l *Logger) Close() error {
	if err := l.Sync(); err != nil {
		return err
	}
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Debug(msg string, fields ...zap.Field) { l.zap.Debug(msg, fields...) }
func (l *Logger) Info(msg string, fields ...zap.Field)  { l.zap.Info(msg, fields...) }
func (l *Logger) Warn(msg string, fields ...zap.Field)  { l.zap.Warn(msg, fields...) }
func (l *Logger) Error(msg string, fields ...zap.Field) { l.zap.Error(msg, fields...) }
func (l *Logger) Fatal(msg string, fields ...zap.Field) { l.zap.Fatal(msg, fields...) }
func (l *Logger) Panic(msg string, fields ...zap.Field) { l.zap.Panic(msg, fields...) }

func (l *Logger) Debugf(template string, args ...any) { l.sugar.Debugf(template, args...) }
func (l *Logger) Infof(template string, args ...any)  { l.sugar.Infof(template, args...) }
func (l *Logger) Warnf(template string, args ...any)  { l.sugar.Warnf(template, args...) }
func (l *Logger) Errorf(template string, args ...any) { l.sugar.Errorf(template, args...) }
func (l *Logger) Fatalf(template string, args ...any) { l.sugar.Fatalf(template, args...) }
func (l *Logger) Panicf(template string, args ...any) { l.sugar.Panicf(template, args...) }

func parseLevel(level string) (zapcore.Level, error) {
	var lv zapcore.Level
	if err := lv.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		return zapcore.InfoLevel, err
	}
	return lv, nil
}

func newEncoder(format string) zapcore.Encoder {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "time"
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.CallerKey = "caller"

	switch strings.ToLower(format) {
	case "console":
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(cfg)
	default:
		return zapcore.NewJSONEncoder(cfg)
	}
}
