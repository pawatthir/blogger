package tests

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/your-username/blogger/logger"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name   string
		config logger.Config
		envs   map[string]string
		want   struct {
			env         string
			serviceName string
			version     string
		}
	}{
		{
			name: "local config",
			config: logger.Config{
				Env:         "local",
				ServiceName: "test-service",
				Level:       "info",
				UseJSON:     true,
			},
			envs: map[string]string{},
			want: struct {
				env         string
				serviceName string
				version     string
			}{
				env:         "local",
				serviceName: "test-service",
				version:     "unknown",
			},
		},
		{
			name: "production config with env overrides",
			config: logger.Config{
				Env:         "local",
				ServiceName: "test-service",
				Level:       "info",
				UseJSON:     true,
			},
			envs: map[string]string{
				"DD_ENV":     "production",
				"DD_SERVICE": "production-service",
				"DD_VERSION": "v1.0.0",
			},
			want: struct {
				env         string
				serviceName string
				version     string
			}{
				env:         "production",
				serviceName: "production-service",
				version:     "v1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envs {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Capture log output
			var buf bytes.Buffer
			originalOutput := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			go func() {
				defer r.Close()
				buf.ReadFrom(r)
			}()

			// Initialize logger
			slogger := logger.Init(tt.config)

			w.Close()
			os.Stdout = originalOutput

			// Verify logger is not nil and is set as default
			assert.NotNil(t, slogger)
			assert.Equal(t, slogger, slog.Default())

			// Verify global variables are set correctly
			assert.Equal(t, tt.want.env, logger.Env)
			assert.Equal(t, tt.want.serviceName, logger.ServiceName)
			assert.Equal(t, tt.want.version, logger.Version)

			// Verify logger works
			assert.NotNil(t, logger.Log)
			assert.NotNil(t, logger.Slog)
		})
	}
}

func TestHandler_AddDDFields(t *testing.T) {
	// Set test environment
	os.Setenv("DD_ENV", "test")
	os.Setenv("DD_SERVICE", "test-service")
	os.Setenv("DD_VERSION", "v1.0.0")
	defer func() {
		os.Unsetenv("DD_ENV")
		os.Unsetenv("DD_SERVICE")
		os.Unsetenv("DD_VERSION")
	}()

	config := logger.Config{
		Env:         "local",
		ServiceName: "blogger",
		Level:       "info",
		UseJSON:     true,
	}

	// Initialize logger
	slogger := logger.Init(config)

	// Test that DD fields are available through global variables
	assert.Equal(t, "test", logger.Env)
	assert.Equal(t, "test-service", logger.ServiceName)
	assert.Equal(t, "v1.0.0", logger.Version)

	// Test that we can log with the initialized logger
	slogger.Info("test message", "key", "value")
}

func TestHandler_WithAttrs(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "test-service",
		Level:       "info",
		UseJSON:     true,
	}

	slogger := logger.Init(config)

	// Test that we can add attributes and log
	loggerWithAttrs := slogger.With(
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	)

	loggerWithAttrs.Info("test message")
}

func TestHandler_WithGroup(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "test-service",
		Level:       "info",
		UseJSON:     true,
	}

	slogger := logger.Init(config)

	// Test that we can create a grouped logger
	groupLogger := slogger.WithGroup("test_group")
	groupLogger.Info("test message", "grouped_key", "grouped_value")
}

func TestHandler_Enabled(t *testing.T) {
	tests := []struct {
		name        string
		configLevel string
		testLevel   slog.Level
		expected    bool
	}{
		{
			name:        "debug level allows debug",
			configLevel: "debug",
			testLevel:   slog.LevelDebug,
			expected:    true,
		},
		{
			name:        "info level blocks debug",
			configLevel: "info",
			testLevel:   slog.LevelDebug,
			expected:    false,
		},
		{
			name:        "info level allows info",
			configLevel: "info",
			testLevel:   slog.LevelInfo,
			expected:    true,
		},
		{
			name:        "warn level allows error",
			configLevel: "warn",
			testLevel:   slog.LevelError,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := logger.Config{
				Env:         "test",
				ServiceName: "test-service",
				Level:       tt.configLevel,
				UseJSON:     true,
			}

			slogger := logger.Init(config)

			// Test by checking if the logger would accept the level
			ctx := context.Background()
			enabled := slogger.Enabled(ctx, tt.testLevel)
			assert.Equal(t, tt.expected, enabled)
		})
	}
}

func TestPathfinder(t *testing.T) {
	// Initialize logger first
	config := logger.Config{
		Env:         "test",
		ServiceName: "test-service",
		Level:       "info",
		UseJSON:     true,
	}
	logger.Init(config)

	pathfinder := logger.NewPathfinder("test-service")
	assert.NotNil(t, pathfinder)

	// Test InfoContext
	ctx := context.Background()
	pathfinder.InfoContext(ctx, "test info message")

	// Test ErrorContext
	pathfinder.ErrorContext(ctx, "test error message")

	// Test NewPathfinder method
	newPF := pathfinder.NewPathfinder("new-service")
	assert.NotNil(t, newPF)
}

func TestGetEnvOrDefault(t *testing.T) {
	// Test with environment variable set
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	// Note: getEnvOrDefault is not exported, so we test it indirectly through Init
	config := logger.Config{
		Env:         "default_env",
		ServiceName: "default_service",
		Level:       "info",
		UseJSON:     true,
	}

	os.Setenv("DD_ENV", "override_env")
	defer os.Unsetenv("DD_ENV")

	logger.Init(config)

	// Verify that environment variable overrode the config
	assert.Equal(t, "override_env", logger.Env)
}

func TestConcurrentInit(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "concurrent-test",
		Level:       "info",
		UseJSON:     true,
	}

	// Test concurrent initialization doesn't cause race conditions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			logger.Init(config)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify logger is still functional
	assert.NotNil(t, logger.Slog)
	assert.Equal(t, "test", logger.Env)
	assert.Equal(t, "concurrent-test", logger.ServiceName)
}
