package logger

import (
	"github.com/sirupsen/logrus"

	"github.com/kakkk/gopkg/requestid"
)

type contextHook struct{}

func (h contextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h contextHook) Fire(entry *logrus.Entry) error {
	if entry.Context != nil {
		requestID := requestid.Get(entry.Context)
		if requestID != "" {
			entry.Data["request_id"] = requestID
		}
	}
	return nil
}
