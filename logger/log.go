package logger

import (
	"log"
	"log/slog"
	"strings"
)

type Logf func(format string, args ...any)

type StdLogger struct {
	Level slog.Level
	Logf  Logf
}

var logger = StdLogger{
	Level: slog.LevelInfo,
	Logf:  log.Printf,
}

func SetUp(level slog.Level, logf Logf) {
	logger.Level = level
	logger.Logf = logf
}

func Errorf(format string, args ...any) {
	if logger.Level > slog.LevelError {
		return
	}
	logger.Logf(format, args...)
}

func Warnf(format string, args ...any) {
	if logger.Level > slog.LevelWarn {
		return
	}
	logger.Logf(format, args...)
}

func Infof(format string, args ...any) {
	if logger.Level > slog.LevelInfo {
		return
	}
	logger.Logf(format, args...)
}

func Debugf(format string, args ...any) {
	if logger.Level > slog.LevelDebug {
		return
	}
	logger.Logf(format, args...)
}

func Error(args ...any) {
	if logger.Level > slog.LevelError {
		return
	}
	fmtStr := strings.Repeat("%v ", len(args))
	logger.Logf(fmtStr, args...)
}
