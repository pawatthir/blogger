package logger

import (
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type CoolEncoder struct {
	zapcore.Encoder
}

func (c *CoolEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	filtered := make([]zapcore.Field, 0, len(fields))
	for _, field := range fields {
		if field.Key == "skip" || field.Type == zapcore.Int64Type {
			continue
		}
		filtered = append(filtered, field)
	}
	return c.Encoder.EncodeEntry(entry, filtered)
}

func newZapLogger(config Config) (*zap.Logger, *slog.Logger) {
	zapLogLevel := getZapLogLevel(config.Level)

	lumberjackLogger := &lumberjack.Logger{
		Filename:   config.FilePath,
		MaxSize:    config.FileSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   true,
	}

	fileWriter := zapcore.AddSync(lumberjackLogger)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	zap.RegisterEncoder("cool", func(config zapcore.EncoderConfig) (zapcore.Encoder, error) {
		return &CoolEncoder{jsonEncoder}, nil
	})

	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	zapCoreList := []zapcore.Core{}
	if config.FileEnabled {
		zapCoreList = append(zapCoreList, zapcore.NewCore(jsonEncoder, fileWriter, zapLogLevel))
	}

	if config.UseJSON {
		zapCoreList = append(zapCoreList, zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), zapLogLevel))
	}

	var core zapcore.Core
	if len(zapCoreList) == 0 {
		core = zapcore.NewTee(zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLogLevel))
	} else {
		core = zapcore.NewTee(zapCoreList...)
	}

	zapLogger := zap.New(core, zap.AddCaller())
	slogLogger := slog.New(NewOtelHandler(zapslog.NewHandler(core, zapslog.WithCaller(true))))

	return zapLogger, slogLogger
}

func getZapLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}