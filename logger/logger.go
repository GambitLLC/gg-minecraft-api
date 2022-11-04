package logger

import (
	"fmt"
	"go.uber.org/zap"
)

type ZapLogger struct {
	Logger *zap.Logger
}

func (z *ZapLogger) Printf(format string, args ...interface{}) {
	z.Logger.Info(fmt.Sprintf(format, args...))
}

func (z *ZapLogger) Debug(format string, args ...interface{}) {
	z.Logger.Debug(fmt.Sprintf(format, args...))
}

func (z *ZapLogger) Info(format string, args ...interface{}) {
	z.Logger.Info(fmt.Sprintf(format, args...))
}

func (z *ZapLogger) Warn(format string, args ...interface{}) {
	z.Logger.Warn(fmt.Sprintf(format, args...))
}

func (z *ZapLogger) Error(format string, args ...interface{}) {
	z.Logger.Error(fmt.Sprintf(format, args...))
}

func (z *ZapLogger) Fatal(format string, args ...interface{}) {
	z.Logger.Fatal(fmt.Sprintf(format, args...))
}

func NewLogger() *ZapLogger {
	zl, err := zap.NewProduction()

	if err != nil {
		panic(err)
	}

	return &ZapLogger{
		Logger: zl,
	}
}
