package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/your-username/blogger/logger"
	"github.com/your-username/blogger/middleware/grpcserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Mock proto message for testing
type mockProtoMessage struct {
	Value string `json:"value"`
}

func (m *mockProtoMessage) Reset()         {}
func (m *mockProtoMessage) String() string { return m.Value }
func (m *mockProtoMessage) ProtoMessage()  {}

// Mock handler function
type mockHandler func(ctx context.Context, req interface{}) (interface{}, error)

func (h mockHandler) Handle(ctx context.Context, req interface{}) (interface{}, error) {
	return h(ctx, req)
}

func TestNewUnaryLoggerInterceptor(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "grpc-test",
		Level:       "info",
		UseJSON:     true,
	}
	slogger := logger.Init(config)

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	assert.NotNil(t, interceptor)

	unaryInterceptor := interceptor.Intercept()
	assert.NotNil(t, unaryInterceptor)
}

func TestLoggerInterceptor_SuccessfulRequest(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "grpc-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	ctx := context.Background()
	req := &wrapperspb.StringValue{Value: "test request"}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return &wrapperspb.StringValue{Value: "test response"}, nil
	}

	resp, err := unaryInterceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	respMsg, ok := resp.(*wrapperspb.StringValue)
	assert.True(t, ok)
	assert.Equal(t, "test response", respMsg.Value)
}

func TestLoggerInterceptor_ErrorRequest(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "grpc-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	ctx := context.Background()
	req := &wrapperspb.StringValue{Value: "test request"}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TestMethod",
	}

	expectedErr := status.Error(codes.NotFound, "resource not found")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	resp, err := unaryInterceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, expectedErr, err)
}

func TestLoggerInterceptor_HealthCheckSkipped(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "grpc-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	ctx := context.Background()
	req := &emptypb.Empty{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return &emptypb.Empty{}, nil
	}

	resp, err := unaryInterceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, handlerCalled, "Handler should be called for health check")
}

func TestProtoMessageToJsonBytes(t *testing.T) {
	tests := []struct {
		name        string
		message     proto.Message
		expectError bool
		expectNil   bool
	}{
		{
			name:        "nil message",
			message:     nil,
			expectError: false,
			expectNil:   true,
		},
		{
			name:        "empty message",
			message:     &emptypb.Empty{},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "string value message",
			message:     &wrapperspb.StringValue{Value: "test"},
			expectError: false,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: protoMessageToJsonBytes is not exported, so we test it indirectly
			// We'll test the behavior through the interceptor
			config := logger.Config{
				Env:         "test",
				ServiceName: "proto-test",
				Level:       "debug",
				UseJSON:     true,
			}
			slogger := logger.Init(config)
			logger.CompileCanonicalLogTemplate()

			interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
			unaryInterceptor := interceptor.Intercept()

			ctx := context.Background()
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.service/TestMethod",
			}

			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return tt.message, nil
			}

			resp, err := unaryInterceptor(ctx, tt.message, info, handler)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNil && !tt.expectError {
				// For nil input, we expect the response to be whatever the handler returns
				assert.Equal(t, tt.message, resp)
			}
		})
	}
}

func TestLoggerInterceptor_TimingMeasurement(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "timing-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	ctx := context.Background()
	req := &wrapperspb.StringValue{Value: "test"}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/SlowMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Simulate slow processing
		time.Sleep(10 * time.Millisecond)
		return &wrapperspb.StringValue{Value: "response"}, nil
	}

	start := time.Now()
	resp, err := unaryInterceptor(ctx, req, info, handler)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

func TestLoggerInterceptor_ContextPropagation(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "context-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	// Create context with a value
	ctx := context.WithValue(context.Background(), "test-key", "test-value")
	req := &wrapperspb.StringValue{Value: "test"}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/ContextMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Verify context is properly propagated
		value := ctx.Value("test-key")
		assert.Equal(t, "test-value", value)
		return &wrapperspb.StringValue{Value: "response"}, nil
	}

	resp, err := unaryInterceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGRPCServerInterceptor_PanicWhenLoggerNotInitialized(t *testing.T) {
	// Save current logger
	originalSlog := logger.Slog
	defer func() {
		logger.Slog = originalSlog
	}()

	// Set logger to nil to simulate uninitialized state
	logger.Slog = nil

	assert.Panics(t, func() {
		grpcserver.GRPCServerInterceptor()
	})
}

func TestLoggerInterceptor_WithNilProtoMessages(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "nil-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	ctx := context.Background()
	req := (*wrapperspb.StringValue)(nil) // nil proto message
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/NilMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return (*wrapperspb.StringValue)(nil), nil
	}

	resp, err := unaryInterceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.Nil(t, resp)
}

func TestLoggerInterceptor_NonProtoMessages(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "non-proto-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	ctx := context.Background()
	req := "not a proto message" // non-proto type
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/NonProtoMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response string", nil
	}

	resp, err := unaryInterceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response string", resp)
}

func TestLoggerInterceptor_DifferentErrorCodes(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "error-codes-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	errorTests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"not found", status.Error(codes.NotFound, "not found"), codes.NotFound},
		{"invalid argument", status.Error(codes.InvalidArgument, "bad request"), codes.InvalidArgument},
		{"internal error", status.Error(codes.Internal, "internal error"), codes.Internal},
		{"permission denied", status.Error(codes.PermissionDenied, "access denied"), codes.PermissionDenied},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &wrapperspb.StringValue{Value: "test"}
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.service/ErrorMethod",
			}

			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, tt.err
			}

			resp, err := unaryInterceptor(ctx, req, info, handler)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tt.err, err)

			// Verify the status code
			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, tt.code, st.Code())
		})
	}
}

func TestLoggerInterceptor_ReflectValueHandling(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "reflect-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	interceptor := grpcserver.NewUnaryLoggerInterceptor(*slogger)
	unaryInterceptor := interceptor.Intercept()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/ReflectMethod",
	}

	// Test with a proto message that has a nil value when reflected
	var nilProto *wrapperspb.StringValue

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nilProto, nil
	}

	resp, err := unaryInterceptor(ctx, nilProto, info, handler)

	assert.NoError(t, err)
	assert.Nil(t, resp)
}
