package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-username/blogger/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestCoolEncoder_EncodeEntry(t *testing.T) {
	// Create a base JSON encoder
	config := zap.NewProductionEncoderConfig()
	jsonEncoder := zapcore.NewJSONEncoder(config)
	coolEncoder := &logger.CoolEncoder{Encoder: jsonEncoder}

	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Time:    time.Now(),
		Message: "test message",
	}

	fields := []zapcore.Field{
		zap.String("keep", "this field should be kept"),
		zap.String("skip", "this field should be removed"),
		zap.Int64("remove_me", 123), // Int64Type should be filtered
		zap.String("keep_me", "456"),     // String type should be kept
	}

	buf, err := coolEncoder.EncodeEntry(entry, fields)
	require.NoError(t, err)
	require.NotNil(t, buf)

	output := buf.String()

	// Verify that "skip" field and Int64 field are removed
	assert.NotContains(t, output, "skip")
	assert.NotContains(t, output, "remove_me")
	assert.NotContains(t, output, "123")

	// Verify that other fields are kept
	assert.Contains(t, output, "keep")
	assert.Contains(t, output, "this field should be kept")
	assert.Contains(t, output, "keep_me")
	assert.Contains(t, output, "456")
}

func TestNewZapLogger(t *testing.T) {
	tests := []struct {
		name   string
		config logger.Config
	}{
		{
			name: "console logger",
			config: logger.Config{
				Env:         "test",
				ServiceName: "test-service",
				Level:       "info",
				UseJSON:     false,
				FileEnabled: false,
			},
		},
		{
			name: "json logger",
			config: logger.Config{
				Env:         "test",
				ServiceName: "test-service",
				Level:       "debug",
				UseJSON:     true,
				FileEnabled: false,
			},
		},
		{
			name: "file logger",
			config: logger.Config{
				Env:         "test",
				ServiceName: "test-service",
				Level:       "warn",
				UseJSON:     true,
				FileEnabled: true,
				FilePath:    "/tmp/test.log",
				FileSize:    100,
				MaxAge:      30,
				MaxBackups:  5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing file
			if tt.config.FileEnabled {
				os.Remove(tt.config.FilePath)
				defer os.Remove(tt.config.FilePath)
			}

			slogger := logger.Init(tt.config)
			require.NotNil(t, slogger)

			// Test that the logger can log messages
			slogger.Info("test message", "key", "value")
			slogger.Debug("debug message")
			slogger.Warn("warning message")
			slogger.Error("error message")

			// If file logging is enabled, verify file exists
			if tt.config.FileEnabled {
				_, err := os.Stat(tt.config.FilePath)
				assert.NoError(t, err, "Log file should exist")
			}
		})
	}
}

func TestGetZapLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zapcore.Level
	}{
		{
			name:     "debug level",
			level:    "debug",
			expected: zap.DebugLevel,
		},
		{
			name:     "info level",
			level:    "info",
			expected: zap.InfoLevel,
		},
		{
			name:     "warn level",
			level:    "warn",
			expected: zap.WarnLevel,
		},
		{
			name:     "error level",
			level:    "error",
			expected: zap.ErrorLevel,
		},
		{
			name:     "panic level",
			level:    "panic",
			expected: zap.PanicLevel,
		},
		{
			name:     "fatal level",
			level:    "fatal",
			expected: zap.FatalLevel,
		},
		{
			name:     "unknown level defaults to info",
			level:    "unknown",
			expected: zap.InfoLevel,
		},
		{
			name:     "empty level defaults to info",
			level:    "",
			expected: zap.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: getZapLogLevel is not exported, so we test it indirectly
			// by creating a logger with that level and checking its behavior
			config := logger.Config{
				Env:         "test",
				ServiceName: "test-service",
				Level:       tt.level,
				UseJSON:     true,
				FileEnabled: false,
			}

			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			var buf bytes.Buffer
			done := make(chan bool)
			go func() {
				buf.ReadFrom(r)
				done <- true
			}()

			slogger := logger.Init(config)

			// Log at different levels
			slogger.Debug("debug message")
			slogger.Info("info message")
			slogger.Warn("warn message")
			slogger.Error("error message")

			w.Close()
			os.Stdout = oldStdout
			<-done

			output := buf.String()

			// Based on the log level, certain messages should or shouldn't appear
			switch tt.expected {
			case zap.DebugLevel:
				assert.Contains(t, output, "debug message")
				assert.Contains(t, output, "info message")
			case zap.InfoLevel:
				assert.NotContains(t, output, "debug message")
				assert.Contains(t, output, "info message")
			case zap.WarnLevel:
				assert.NotContains(t, output, "debug message")
				assert.NotContains(t, output, "info message")
				assert.Contains(t, output, "warn message")
			case zap.ErrorLevel:
				assert.NotContains(t, output, "debug message")
				assert.NotContains(t, output, "info message")
				assert.NotContains(t, output, "warn message")
				assert.Contains(t, output, "error message")
			}
		})
	}
}

func TestFileRotationConfig(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "rotation-test",
		Level:       "info",
		UseJSON:     true,
		FileEnabled: true,
		FilePath:    "/tmp/rotation_test.log",
		FileSize:    1, // 1MB
		MaxAge:      7, // 7 days
		MaxBackups:  3, // 3 backup files
	}

	defer os.Remove(config.FilePath)

	slogger := logger.Init(config)
	require.NotNil(t, slogger)

	// Write multiple log entries to test rotation setup
	for i := 0; i < 100; i++ {
		slogger.Info("test message", "iteration", i)
	}

	// Verify log file exists
	_, err := os.Stat(config.FilePath)
	assert.NoError(t, err)

	// Read the log file and verify it contains JSON logs
	content, err := os.ReadFile(config.FilePath)
	assert.NoError(t, err)

	// Verify JSON format
	lines := bytes.Split(content, []byte("\n"))
	for _, line := range lines {
		if len(line) > 0 {
			var logEntry map[string]interface{}
			err := json.Unmarshal(line, &logEntry)
			assert.NoError(t, err, "Each log line should be valid JSON")

			// Verify required fields exist
			assert.Contains(t, logEntry, "msg")
			assert.Contains(t, logEntry, "level")
			assert.Contains(t, logEntry, "ts")
		}
	}
}

func TestZapLoggerWithOtelHandler(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "otel-test",
		Level:       "info",
		UseJSON:     true,
		FileEnabled: false,
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		buf.ReadFrom(r)
		done <- true
	}()

	slogger := logger.Init(config)
	slogger.Info("test message with otel", "key", "value")

	w.Close()
	os.Stdout = oldStdout
	<-done

	output := buf.String()

	// Verify the log contains the message and structured data
	assert.Contains(t, output, "test message with otel")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	// Verify JSON structure
	lines := bytes.Split([]byte(output), []byte("\n"))
	for _, line := range lines {
		if len(line) > 0 {
			var logEntry map[string]interface{}
			err := json.Unmarshal(line, &logEntry)
			if err == nil { // Some lines might not be JSON (like initialization messages)
				assert.Contains(t, logEntry, "msg")
				assert.Contains(t, logEntry, "level")
			}
		}
	}
}
