package logger

import (
	"log/slog"
	"os"
)

const devLevel = "dev"

func SetupLogger(env string) *slog.Logger {

	logLevel := &slog.LevelVar{}
	opt := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}

	var handler slog.Handler
	if env == devLevel {
		logLevel.Set(slog.LevelDebug)
		handler = slog.NewTextHandler(os.Stdout, opt)
	} else {
		logLevel.Set(slog.LevelInfo)
		handler = slog.NewJSONHandler(os.Stdout, opt)
	}

	return slog.New(handler)
}
