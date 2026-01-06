package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// consoleHook 控制台输出的Hook
type consoleHook struct {
	formatter logrus.Formatter
}

func (hook *consoleHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *consoleHook) Fire(entry *logrus.Entry) error {
	line, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}

	os.Stdout.Write(line)
	return nil
}
