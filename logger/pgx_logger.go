package logger

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/tracelog"
	"go.uber.org/zap"
)

type PGXLogger struct {
	logger *zap.Logger
}

func NewPGXLogger(logger *zap.Logger) *PGXLogger {
	return &PGXLogger{logger: logger.WithOptions(zap.AddCallerSkip(1))}
}

func (pl *PGXLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	fields := make([]any, 0, len(data))
	for k, v := range data {
		fields = append(fields, slog.Any(k, v))
	}

	switch level {
	case tracelog.LogLevelTrace:
		slog.DebugContext(ctx, msg, fields...)
	case tracelog.LogLevelDebug:
		slog.DebugContext(ctx, msg, fields...)
	case tracelog.LogLevelInfo:
		slog.InfoContext(ctx, msg, fields...)
	case tracelog.LogLevelWarn:
		slog.WarnContext(ctx, msg, fields...)
	case tracelog.LogLevelError:
		slog.ErrorContext(ctx, msg, fields...)
	default:
		slog.ErrorContext(ctx, msg, fields...)
	}
}

func NewPGXLoggerFromSlog() *PGXLogger {
	return &PGXLogger{logger: zap.NewNop()}
}