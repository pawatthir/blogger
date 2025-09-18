package grpcserver

import (
	"context"
	"log/slog"
	"reflect"
	"time"

	"github.com/your-username/blogger/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type LoggerInterceptor interface {
	Intercept() grpc.UnaryServerInterceptor
}

type loggerInterceptor struct {
	logger slog.Logger
}

func NewUnaryLoggerInterceptor(slogger slog.Logger) LoggerInterceptor {
	loggerWithName := slogger.With(slog.String("logger_name", "grpc_interceptor"))
	return &loggerInterceptor{
		logger: *loggerWithName,
	}
}

func (l *loggerInterceptor) Intercept() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/grpc.health.v1.Health/Check" {
			return handler(ctx, req)
		}

		startTime := time.Now()

		reqProto, _ := req.(proto.Message)
		requestBody, _ := protoMessageToJsonBytes(reqProto)

		resp, err := handler(ctx, req)
		elapse := time.Since(startTime)
		respProto, _ := resp.(proto.Message)
		responseBody, _ := protoMessageToJsonBytes(respProto)

		var fields []any
		fields = append(fields,
			slog.String("logger_name", "canonical"),
			slog.Group("grpcserver_md",
				slog.String("type", "grpcserver"),
				slog.String("method", "POST"),
				slog.String("path", info.FullMethod),
				slog.String("duration", elapse.String()),
			),
		)

		var level logger.Level
		if err != nil {
			level = logger.Error
		} else {
			level = logger.Info
		}

		logger.CanonicalLogger(
			ctx,
			l.logger,
			level,
			requestBody,
			responseBody,
			err,
			logger.CanonicalLog{
				Transport: "grpc",
				Traffic:   "internal",
				Method:    "POST",
				Status:    int(status.Code(err)),
				Path:      info.FullMethod,
				Duration:  elapse,
			},
			fields,
		)
		return resp, err
	}
}

func protoMessageToJsonBytes(message proto.Message) ([]byte, error) {
	if message == nil || reflect.ValueOf(message).IsNil() {
		return nil, nil
	}

	m := protojson.MarshalOptions{EmitUnpopulated: true}
	jsonBytes, err := m.Marshal(message)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func GRPCServerInterceptor() grpc.UnaryServerInterceptor {
	if logger.Slog == nil {
		panic("Logger not initialized. Call logger.Init() first.")
	}
	return NewUnaryLoggerInterceptor(*logger.Slog).Intercept()
}