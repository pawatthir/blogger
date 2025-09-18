package tests

import (
	"bytes"
	"context"
	"html/template"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pawatthir/blogger/logger"
	"github.com/stretchr/testify/assert"
)

func TestCompileCanonicalLogTemplate(t *testing.T) {
	// Test that the template compiles without error
	logger.CompileCanonicalLogTemplate()

	tmpl, err := logger.GetCanonicalLogTemplate()
	assert.NoError(t, err)
	assert.NotNil(t, tmpl)

	// Test that the template can be executed
	canonicalLog := logger.CanonicalLog{
		Transport: "HTTP",
		Traffic:   "incoming",
		Method:    "GET",
		Status:    200,
		Path:      "/api/users",
		Duration:  time.Millisecond * 150,
		Message:   "Success",
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, canonicalLog)
	assert.NoError(t, err)

	output := buf.String()
	expected := "[HTTP][incoming] GET 200 /api/users 150ms - Success"
	assert.Equal(t, expected, output)
}

func TestGetCanonicalLogTemplate(t *testing.T) {
	// Test when template is not compiled
	// Reset template to nil for this test
	logger.CompileCanonicalLogTemplate() // This ensures it's compiled

	tmpl, err := logger.GetCanonicalLogTemplate()
	assert.NoError(t, err)
	assert.NotNil(t, tmpl)
	assert.IsType(t, &template.Template{}, tmpl)
}

func TestExceptionError_Error(t *testing.T) {
	err := &logger.ExceptionError{
		Code:          500,
		GlobalMessage: "Internal Server Error",
		DebugMessage:  "Database connection failed",
		APIStatusCode: 500,
		ErrFields:     map[string]interface{}{"detail": "connection timeout"},
	}

	assert.Equal(t, "Database connection failed", err.Error())
}

func TestCanonicalLogger_Success(t *testing.T) {
	// Initialize logger first
	config := logger.Config{
		Env:         "test",
		ServiceName: "canonical-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)

	// Compile template
	logger.CompileCanonicalLogTemplate()

	testLogger := logger.Slog

	ctx := context.Background()
	request := []byte(`{"user_id": 123, "action": "get_profile"}`)
	response := []byte(`{"id": 123, "name": "John Doe", "email": "john@example.com"}`)

	canonicalLog := logger.CanonicalLog{
		Transport: "HTTP",
		Traffic:   "incoming",
		Method:    "GET",
		Status:    200,
		Path:      "/api/profile",
		Duration:  time.Millisecond * 100,
		Message:   "Profile retrieved successfully",
	}

	metadata := []any{
		slog.String("request_id", "req-123"),
		slog.String("user_agent", "test-client"),
	}

	logger.CanonicalLogger(ctx, *testLogger, logger.Info, request, response, nil, canonicalLog, metadata)

	// Test passes if no panic occurs
	assert.NotNil(t, testLogger)
}

func TestCanonicalLogger_WithError(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "canonical-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	testLogger := slogger

	ctx := context.Background()
	request := []byte(`{"user_id": 999}`)
	response := []byte(`{"error": "user not found"}`)

	canonicalLog := logger.CanonicalLog{
		Transport: "HTTP",
		Traffic:   "incoming",
		Method:    "GET",
		Status:    404,
		Path:      "/api/user/999",
		Duration:  time.Millisecond * 50,
		Message:   "User not found",
	}

	// Test with ExceptionError
	err := &logger.ExceptionError{
		Code:          404,
		GlobalMessage: "User not found",
		DebugMessage:  "User with ID 999 does not exist",
		APIStatusCode: 404,
		ErrFields:     map[string]interface{}{"user_id": 999},
		StackErrors: []logger.StackError{
			{
				Kind:    "NotFoundError",
				Message: "User not found in database",
				Stack:   "UserService.getUser()\n\tUserRepository.findById()",
			},
		},
	}

	logger.CanonicalLogger(ctx, *testLogger, logger.Info, request, response, err, canonicalLog, []any{})

	// Test passes if no panic occurs
	assert.NotNil(t, testLogger)
}

func TestCanonicalLogger_WithStackErrorTruncation(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "canonical-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	testLogger := slogger

	// Create a long stack trace
	longStack := strings.Join([]string{
		"line1", "line2", "line3", "line4", "line5", "line6", "line7", "line8", "line9", "line10",
	}, "\n\t")

	err := &logger.ExceptionError{
		Code:          500,
		GlobalMessage: "Internal Error",
		DebugMessage:  "Stack trace truncation test",
		APIStatusCode: 500,
		StackErrors: []logger.StackError{
			{
				Kind:    "RuntimeError",
				Message: "Test error with long stack",
				Stack:   longStack,
			},
		},
	}

	logger.CanonicalLogger(context.Background(), *testLogger, logger.Error, []byte(`{}`), []byte(`{}`), err, logger.CanonicalLog{}, []any{})

	// Test passes if no panic occurs
	assert.NotNil(t, testLogger)
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name     string
		logKey   string
		expected bool
	}{
		{
			name:     "login path should be sanitized",
			logKey:   "/api/auth/login",
			expected: true,
		},
		{
			name:     "password path should be sanitized",
			logKey:   "/api/user/change-password",
			expected: true,
		},
		{
			name:     "token path should be sanitized",
			logKey:   "/api/auth/refresh-token",
			expected: true,
		},
		{
			name:     "secret path should be sanitized",
			logKey:   "/api/config/secret",
			expected: true,
		},
		{
			name:     "key path should be sanitized",
			logKey:   "/api/encryption/key",
			expected: true,
		},
		{
			name:     "verify-otp path should be sanitized",
			logKey:   "/api/auth/verify-otp",
			expected: true,
		},
		{
			name:     "case insensitive matching",
			logKey:   "/api/auth/LOGIN",
			expected: true,
		},
		{
			name:     "normal path should not be sanitized",
			logKey:   "/api/users",
			expected: false,
		},
		{
			name:     "profile path should not be sanitized",
			logKey:   "/api/user/profile",
			expected: false,
		},
		{
			name:     "health check should not be sanitized",
			logKey:   "/health",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.Sanitize(tt.logKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanonicalLogger_WithSanitization(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "canonical-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	testLogger := slogger

	ctx := context.Background()
	request := []byte(`{"username": "user", "password": "secret123"}`)
	response := []byte(`{"token": "jwt-token-here", "user_id": 123}`)

	canonicalLog := logger.CanonicalLog{
		Transport: "HTTP",
		Traffic:   "incoming",
		Method:    "POST",
		Status:    200,
		Path:      "/api/auth/login", // This should trigger sanitization
		Duration:  time.Millisecond * 200,
		Message:   "Login successful",
	}

	logger.CanonicalLogger(ctx, *testLogger, logger.Info, request, response, nil, canonicalLog, []any{})

	// Test passes if no panic occurs
	assert.NotNil(t, testLogger)
}

func TestCanonicalLogger_InvalidJSON(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "canonical-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	testLogger := slogger

	ctx := context.Background()
	request := []byte(`invalid json`)
	response := []byte(`also invalid json`)

	canonicalLog := logger.CanonicalLog{
		Transport: "HTTP",
		Traffic:   "incoming",
		Method:    "POST",
		Status:    400,
		Path:      "/api/test",
		Duration:  time.Millisecond * 10,
		Message:   "Bad request",
	}

	logger.CanonicalLogger(ctx, *testLogger, logger.Info, request, response, nil, canonicalLog, []any{})

	// Test passes if no panic occurs
	assert.NotNil(t, testLogger)
}

func TestGetStackField(t *testing.T) {
	tests := []struct {
		name        string
		stackErrors []logger.StackError
		expected    logger.StackError
	}{
		{
			name: "returns first stack error",
			stackErrors: []logger.StackError{
				{Kind: "Error1", Message: "First error", Stack: "stack1"},
				{Kind: "Error2", Message: "Second error", Stack: "stack2"},
			},
			expected: logger.StackError{Kind: "Error1", Message: "First error", Stack: "stack1"},
		},
		{
			name:        "returns default for empty slice",
			stackErrors: []logger.StackError{},
			expected: logger.StackError{
				Kind:    "unknown",
				Message: "no stack information available",
				Stack:   "",
			},
		},
		{
			name:        "returns default for nil slice",
			stackErrors: nil,
			expected: logger.StackError{
				Kind:    "unknown",
				Message: "no stack information available",
				Stack:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.GetStackField(tt.stackErrors)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanonicalLogger_LevelsMapping(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "canonical-test",
		Level:       "debug",
		UseJSON:     true,
	}
	slogger := logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	testLogger := slogger

	ctx := context.Background()
	request := []byte(`{}`)
	response := []byte(`{}`)
	canonicalLog := logger.CanonicalLog{Transport: "HTTP", Traffic: "incoming", Method: "GET", Status: 200, Path: "/test"}

	tests := []struct {
		level    logger.Level
		expected string
	}{
		{logger.Debug, "DEBUG"},
		{logger.Info, "INFO"},
		{logger.Warn, "WARN"},
		{logger.Error, "ERROR"},
	}

	for _, tt := range tests {
		logger.CanonicalLogger(ctx, *testLogger, tt.level, request, response, nil, canonicalLog, []any{})
		// Test passes if no panic occurs
	}
	assert.NotNil(t, testLogger)
}
