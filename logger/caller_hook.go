package logger

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

type callerHook struct {
}

func newCallerHook() *callerHook {
	return &callerHook{}
}

func (h *callerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *callerHook) Fire(entry *logrus.Entry) error {
	frame := h.findCaller()
	if frame != nil {
		entry.Data["file"] = fmt.Sprintf("%s:%d", frame.File, frame.Line)
	}
	return nil
}

func (h *callerHook) findCaller() *runtime.Frame {
	// 遍历调用栈，找到第一个不在 logger 包和 logrus 包中的调用者
	pcs := make([]uintptr, 25)
	n := runtime.Callers(4, pcs)
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if !h.isLoggerPackage(frame.Function) &&
			!strings.Contains(frame.Function, "sirupsen/logrus") {
			return &frame
		}
		if !more {
			break
		}
	}
	return nil
}

func (h *callerHook) isLoggerPackage(funcName string) bool {
	pkg := "github.com/kakkk/gopkg/logger"
	return strings.HasPrefix(funcName, pkg+".") || strings.HasPrefix(funcName, pkg+"/")
}
