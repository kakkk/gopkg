package logger

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

func Ctx(ctx context.Context) *logrus.Entry {
	return globalLogger.WithContext(ctx)
}

func WithError(err error) *logrus.Entry {
	return globalLogger.WithField(logrus.ErrorKey, err)
}

func WithContext(ctx context.Context) *logrus.Entry {
	return globalLogger.WithContext(ctx)
}

func WithField(key string, value interface{}) *logrus.Entry {
	return globalLogger.WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return globalLogger.WithFields(fields)
}

func WithTime(t time.Time) *logrus.Entry {
	return globalLogger.WithTime(t)
}

func Trace(args ...interface{}) {
	globalLogger.Trace(args...)
}

func Debug(args ...interface{}) {
	globalLogger.Debug(args...)
}

func Print(args ...interface{}) {
	globalLogger.Print(args...)
}

func Info(args ...interface{}) {
	globalLogger.Info(args...)
}

func Warn(args ...interface{}) {
	globalLogger.Warn(args...)
}

func Warning(args ...interface{}) {
	globalLogger.Warning(args...)
}

func Error(args ...interface{}) {
	globalLogger.Error(args...)
}

func Panic(args ...interface{}) {
	globalLogger.Panic(args...)
}

func Fatal(args ...interface{}) {
	globalLogger.Fatal(args...)
}

func TraceFn(fn logrus.LogFunction) {
	globalLogger.TraceFn(fn)
}

func DebugFn(fn logrus.LogFunction) {
	globalLogger.DebugFn(fn)
}

func PrintFn(fn logrus.LogFunction) {
	globalLogger.PrintFn(fn)
}

func InfoFn(fn logrus.LogFunction) {
	globalLogger.InfoFn(fn)
}

func WarnFn(fn logrus.LogFunction) {
	globalLogger.WarnFn(fn)
}

func WarningFn(fn logrus.LogFunction) {
	globalLogger.WarningFn(fn)
}

func ErrorFn(fn logrus.LogFunction) {
	globalLogger.ErrorFn(fn)
}

func PanicFn(fn logrus.LogFunction) {
	globalLogger.PanicFn(fn)
}

func FatalFn(fn logrus.LogFunction) {
	globalLogger.FatalFn(fn)
}

func Tracef(format string, args ...interface{}) {
	globalLogger.Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	globalLogger.Debugf(format, args...)
}

func Printf(format string, args ...interface{}) {
	globalLogger.Printf(format, args...)
}

func Infof(format string, args ...interface{}) {
	globalLogger.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	globalLogger.Warnf(format, args...)
}

func Warningf(format string, args ...interface{}) {
	globalLogger.Warningf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	globalLogger.Errorf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	globalLogger.Panicf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	globalLogger.Fatalf(format, args...)
}

func Traceln(args ...interface{}) {
	globalLogger.Traceln(args...)
}

func Debugln(args ...interface{}) {
	globalLogger.Debugln(args...)
}

func Println(args ...interface{}) {
	globalLogger.Println(args...)
}

func Infoln(args ...interface{}) {
	globalLogger.Infoln(args...)
}

func Warnln(args ...interface{}) {
	globalLogger.Warnln(args...)
}

func Warningln(args ...interface{}) {
	globalLogger.Warningln(args...)
}

func Errorln(args ...interface{}) {
	globalLogger.Errorln(args...)
}

func Panicln(args ...interface{}) {
	globalLogger.Panicln(args...)
}

func Fatalln(args ...interface{}) {
	globalLogger.Fatalln(args...)
}
