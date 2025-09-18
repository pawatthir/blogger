package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/go-slog/otelslog"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	Log  *zap.Logger
	Slog *slog.Logger
	m    sync.Mutex
)

var Env, ServiceName, Version string

type Field = zap.Field

type Config struct {
	Env         string
	ServiceName string
	Level       string
	UseJSON     bool
	FileEnabled bool
	FilePath    string
	FileSize    int
	MaxAge      int
	MaxBackups  int
}

func Init(config Config) *slog.Logger {
	m.Lock()
	defer m.Unlock()

	Env = getEnvOrDefault("DD_ENV", config.Env)
	ServiceName = getEnvOrDefault("DD_SERVICE", config.ServiceName)
	Version = getEnvOrDefault("DD_VERSION", "unknown")

	// Create the zap logger
	zapLogger, slogLogger := newZapLogger(config)
	Log = zapLogger
	Slog = slogLogger
	slog.SetDefault(Slog)
	CompileCanonicalLogTemplate()
	slog.InfoContext(context.Background(), "Logger initialized")

	return Slog
}

var _ slog.Handler = Handler{}

type Handler struct {
	handler slog.Handler
}

func NewOtelHandler(handler slog.Handler) Handler {
	return Handler{handler: otelslog.NewHandler(handler)}
}

func NewHandler(handler slog.Handler) Handler {
	return Handler{handler: handler}
}

func (h Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h Handler) Handle(ctx context.Context, record slog.Record) error {
	AddDDFields(ctx, &record)
	return h.handler.Handle(ctx, record)
}

func AddDDFields(ctx context.Context, record *slog.Record) {
	spanCtx := trace.SpanContextFromContext(ctx)
	var traceID, spanID string

	if spanCtx.HasTraceID() {
		traceID = spanCtx.TraceID().String()
		record.AddAttrs(slog.String("trace_id", traceID))
	}

	if spanCtx.HasSpanID() {
		spanID = spanCtx.SpanID().String()
		record.AddAttrs(slog.String("span_id", spanID))
	}

	record.AddAttrs(slog.Group("dd",
		slog.String("env", Env),
		slog.String("service", ServiceName),
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
		slog.String("version", Version),
	))
}

func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Handler{h.handler.WithAttrs(attrs)}
}

func (h Handler) WithGroup(name string) slog.Handler {
	return h.handler.WithGroup(name)
}

func getDefaultConfig(env string) Config {
	if env == "local" || env == "development" {
		return Config{
			Env:         env,
			ServiceName: ServiceName,
			Level:       "debug",
			UseJSON:     false,
			FileEnabled: false,
			FilePath:    "logs/app.log",
			FileSize:    100,
			MaxAge:      30,
			MaxBackups:  3,
		}
	}

	return Config{
		Env:         env,
		ServiceName: ServiceName,
		Level:       "info",
		UseJSON:     true,
		FileEnabled: false,
		FilePath:    "logs/app.log",
		FileSize:    100,
		MaxAge:      30,
		MaxBackups:  3,
	}
}

type Pathfinder struct {
	svc string
}

func NewPathfinder(svc string) Pathfinder {
	return Pathfinder{svc: svc}
}

func (p Pathfinder) InfoContext(ctx context.Context, msg string, fields ...any) {
	slog.InfoContext(ctx, "[service]["+p.svc+"] "+msg, fields...)
}

func (p Pathfinder) ErrorContext(ctx context.Context, msg string, fields ...any) {
	slog.ErrorContext(ctx, "[service]["+p.svc+"] "+msg, fields...)
}

func (p Pathfinder) NewPathfinder(svc string) Pathfinder {
	return Pathfinder{svc: p.svc + "." + svc}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}