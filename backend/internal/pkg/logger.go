package pkg

import (
	"io"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type StructuredLogger struct {
	logger log.Logger
}

func NewStructuredLogger(serviceName string) *StructuredLogger {
	writer := io.Writer(os.Stdout)

	if err := os.MkdirAll("logs", 0o755); err == nil {
		if file, openErr := os.OpenFile("logs/backend.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); openErr == nil {
			writer = io.MultiWriter(os.Stdout, file)
		}
	}

	base := log.NewJSONLogger(log.NewSyncWriter(writer))
	base = log.With(
		base,
		"ts", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
		"service", serviceName,
	)

	return &StructuredLogger{logger: base}
}

func (l *StructuredLogger) With(keyvals ...interface{}) *StructuredLogger {
	return &StructuredLogger{
		logger: log.With(l.logger, keyvals...),
	}
}

func (l *StructuredLogger) Info(msg string, keyvals ...interface{}) {
	_ = level.Info(l.logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
}

func (l *StructuredLogger) Debug(msg string, keyvals ...interface{}) {
	_ = level.Debug(l.logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
}

func (l *StructuredLogger) Warn(msg string, keyvals ...interface{}) {
	_ = level.Warn(l.logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
}

func (l *StructuredLogger) Error(msg string, keyvals ...interface{}) {
	_ = level.Error(l.logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
}
