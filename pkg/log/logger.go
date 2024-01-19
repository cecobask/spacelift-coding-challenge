package log

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func NewLogger(writer io.Writer, level slog.Leveler) *Logger {
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})
	return &Logger{
		slog.New(handler),
	}
}

func DefaultLogger() *Logger {
	return NewLogger(os.Stdout, slog.LevelDebug)
}

func (l *Logger) ExitOnError(err error) {
	if err != nil {
		l.Error(err.Error())
		os.Exit(1)
	}
}

type contextKey string

const loggerKey contextKey = "logger"

func WithContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *Logger {
	logger, ok := ctx.Value(loggerKey).(*Logger)
	if !ok {
		return DefaultLogger()
	}
	return logger
}
