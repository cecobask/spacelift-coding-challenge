package log

import (
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
