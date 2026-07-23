package logs

import "go.uber.org/zap"

// 包级便捷方法，项目内统一通过 logs.Info / logs.Infof 等方式打日志。

func Debug(msg string, fields ...zap.Field) { Default().Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)  { Default().Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { Default().Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { Default().Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { Default().Fatal(msg, fields...) }
func Panic(msg string, fields ...zap.Field) { Default().Panic(msg, fields...) }

func Debugf(template string, args ...any) { Default().Debugf(template, args...) }
func Infof(template string, args ...any)  { Default().Infof(template, args...) }
func Warnf(template string, args ...any)  { Default().Warnf(template, args...) }
func Errorf(template string, args ...any) { Default().Errorf(template, args...) }
func Fatalf(template string, args ...any) { Default().Fatalf(template, args...) }
func Panicf(template string, args ...any) { Default().Panicf(template, args...) }

// Sync 刷盘全局日志缓冲。
func Sync() error { return Default().Sync() }

// Close 关闭全局日志。
func Close() error { return Default().Close() }
