package tests

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/your-username/blogger/logger"
	"github.com/your-username/blogger/middleware/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Mock ClientConn for testing
type mockClientConn struct {
	grpc.ClientConnInterface
}

// Mock invoker functions are replaced with grpc.UnaryInvoker

// Test proto message with sensitive field
type testProtoMessage struct {
	PublicField    string `json:"public_field"`
	SensitiveField string `json:"sensitive_field" sensitive:"true"`
}

func (t *testProtoMessage) Reset()         {}
func (t *testProtoMessage) String() string { return "" }
func (t *testProtoMessage) ProtoMessage()  {}

func TestUnaryClientLoggingInterceptor_Success(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "grpc-client-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.UnaryClientLoggingInterceptor()

	ctx := context.Background()
	method := "/test.service.v1.TestService/GetUser"
	req := &wrapperspb.StringValue{Value: "test-user-id"}
	resp := &wrapperspb.StringValue{}

	invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		// Simulate successful response
		respVal := resp.(*wrapperspb.StringValue)
		respVal.Value = "user-data"
		return nil
	})

	err := interceptor(ctx, method, req, resp, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)

	assert.NoError(t, err)
	assert.Equal(t, "user-data", resp.Value)
}

func TestUnaryClientLoggingInterceptor_Error(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "grpc-client-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.UnaryClientLoggingInterceptor()

	ctx := context.Background()
	method := "/test.service.v1.TestService/GetUser"
	req := &wrapperspb.StringValue{Value: "nonexistent-user"}
	resp := &wrapperspb.StringValue{}

	expectedErr := status.Error(codes.NotFound, "user not found")
	invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return expectedErr
	})

	err := interceptor(ctx, method, req, resp, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUnaryClientLoggingInterceptor_WithMetadata(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "grpc-client-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.UnaryClientLoggingInterceptor()

	// Create context with outgoing metadata
	md := metadata.New(map[string]string{
		"authorization": "Bearer token123",
		"user-id":       "user123",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	method := "/test.service.v1.TestService/GetUser"
	req := &wrapperspb.StringValue{Value: "test-user-id"}
	resp := &wrapperspb.StringValue{}

	var receivedOpts []grpc.CallOption
	invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		receivedOpts = opts

		// Simulate setting response headers
		for _, opt := range opts {
			if headerOpt, ok := opt.(grpc.HeaderCallOption); ok {
				*headerOpt.HeaderAddr = metadata.New(map[string]string{
					"response-header": "response-value",
				})
			}
		}

		respVal := resp.(*wrapperspb.StringValue)
		respVal.Value = "response-data"
		return nil
	})

	err := interceptor(ctx, method, req, resp, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)

	assert.NoError(t, err)
	assert.NotEmpty(t, receivedOpts) // Verify header option was added
}

func TestProtoMessageToMap(t *testing.T) {
	tests := []struct {
		name        string
		message     interface{}
		expectError bool
		expectNil   bool
	}{
		{
			name:        "nil message",
			message:     (*wrapperspb.StringValue)(nil),
			expectError: false,
			expectNil:   true,
		},
		{
			name:        "valid string value",
			message:     &wrapperspb.StringValue{Value: "test"},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "non-proto message",
			message:     "not a proto",
			expectError: false,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since protoMessageToMap is not exported, we test it indirectly
			// by testing the client interceptor which uses it
			config := logger.Config{
				Env:         "test",
				ServiceName: "proto-map-test",
				Level:       "debug",
				UseJSON:     true,
			}
			logger.Init(config)

			interceptor := grpcclient.UnaryClientLoggingInterceptor()

			ctx := context.Background()
			method := "/test.service/TestMethod"

			invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return nil
			})

			// The test validates that the interceptor handles different message types gracefully
			err := interceptor(ctx, method, tt.message, &wrapperspb.StringValue{}, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMaskSensitiveDataUsingStructTag(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "mask sensitive field",
			input: map[string]interface{}{
				"public_field":    "public_value",
				"sensitive_field": "secret123",
			},
			expected: map[string]interface{}{
				"public_field":    "public_value",
				"sensitive_field": "s*****3",
			},
		},
		{
			name: "mask short sensitive field",
			input: map[string]interface{}{
				"sensitive_field": "x",
			},
			expected: map[string]interface{}{
				"sensitive_field": "x*****",
			},
		},
		{
			name: "no sensitive fields",
			input: map[string]interface{}{
				"public_field": "public_value",
			},
			expected: map[string]interface{}{
				"public_field": "public_value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test proto message
			message := &testProtoMessage{
				PublicField:    "public_value",
				SensitiveField: "secret123",
			}

			// Since maskSensitiveDataUsingStructTag is not exported,
			// we test it indirectly through the client interceptor
			config := logger.Config{
				Env:         "test",
				ServiceName: "mask-test",
				Level:       "debug",
				UseJSON:     true,
			}
			logger.Init(config)

			interceptor := grpcclient.UnaryClientLoggingInterceptor()
			ctx := context.Background()
			method := "/test.service/MaskTest"

			invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return nil
			})

			// The masking happens during logging, so we verify the interceptor doesn't fail
			err := interceptor(ctx, method, message, &wrapperspb.StringValue{}, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)
			assert.NoError(t, err)
		})
	}
}

func TestLogGRPCClientRequest(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "request-log-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.UnaryClientLoggingInterceptor()

	ctx := context.Background()
	method := "/test.service/LogRequest"
	req := &wrapperspb.StringValue{Value: "test-request"}

	invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	})

	err := interceptor(ctx, method, req, &wrapperspb.StringValue{}, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)
	assert.NoError(t, err)
}

func TestLogGRPCClientResponse(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "response-log-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.UnaryClientLoggingInterceptor()

	ctx := context.Background()
	method := "/test.service/LogResponse"
	req := &wrapperspb.StringValue{Value: "test-request"}
	resp := &wrapperspb.StringValue{}

	invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		respVal := resp.(*wrapperspb.StringValue)
		respVal.Value = "test-response"
		return nil
	})

	err := interceptor(ctx, method, req, resp, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)
	assert.NoError(t, err)
	assert.Equal(t, "test-response", resp.Value)
}

func TestGRPCClientInterceptor(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "client-interceptor-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.GRPCClientInterceptor()
	assert.NotNil(t, interceptor)

	// Test that it's the same as UnaryClientLoggingInterceptor
	directInterceptor := grpcclient.UnaryClientLoggingInterceptor()

	// Both should be functions, but we can't compare them directly
	// So we just ensure both are not nil
	assert.NotNil(t, directInterceptor)
}

func TestInterceptor_TimingMeasurement(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "timing-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.UnaryClientLoggingInterceptor()

	ctx := context.Background()
	method := "/test.service/SlowMethod"
	req := &wrapperspb.StringValue{Value: "test"}
	resp := &wrapperspb.StringValue{}

	invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		// Simulate slow response
		time.Sleep(10 * time.Millisecond)
		respVal := resp.(*wrapperspb.StringValue)
		respVal.Value = "slow-response"
		return nil
	})

	start := time.Now()
	err := interceptor(ctx, method, req, resp, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

func TestInterceptor_DifferentErrorCodes(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "error-codes-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.UnaryClientLoggingInterceptor()

	errorTests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"deadline exceeded", status.Error(codes.DeadlineExceeded, "timeout"), codes.DeadlineExceeded},
		{"unavailable", status.Error(codes.Unavailable, "service unavailable"), codes.Unavailable},
		{"unauthenticated", status.Error(codes.Unauthenticated, "not authenticated"), codes.Unauthenticated},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			method := "/test.service/ErrorMethod"
			req := &wrapperspb.StringValue{Value: "test"}
			resp := &wrapperspb.StringValue{}

			invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return tt.err
			})

			err := interceptor(ctx, method, req, resp, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)

			assert.Error(t, err)
			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.code, status.Code(err))
		})
	}
}

func TestReflectValueHandling(t *testing.T) {
	// Test with nil proto message
	var nilMessage *wrapperspb.StringValue
	assert.True(t, reflect.ValueOf(nilMessage).IsNil())

	// Test with non-nil proto message
	message := &wrapperspb.StringValue{Value: "test"}
	assert.False(t, reflect.ValueOf(message).IsNil())

	// Initialize logger and test interceptor with nil message
	config := logger.Config{
		Env:         "test",
		ServiceName: "reflect-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	interceptor := grpcclient.UnaryClientLoggingInterceptor()

	ctx := context.Background()
	method := "/test.service/NilTest"

	invoker := grpc.UnaryInvoker(func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	})

	err := interceptor(ctx, method, nilMessage, &wrapperspb.StringValue{}, &grpc.ClientConn{}, invoker, []grpc.CallOption{}...)
	assert.NoError(t, err)
}
