package tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-username/blogger/config"
)

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		want     *config.LogConfig
		wantErr  bool
	}{
		{
			name: "valid config",
			yamlData: `log:
  env: production
  serviceName: test-service
  level: info
  useJsonEncoder: true
  fileEnabled: true
  filePath: ./logs/app.log
  fileSize: 100
  maxAge: 30
  maxBackups: 5`,
			want: &config.LogConfig{
				Env:         "production",
				ServiceName: "test-service",
				Level:       "info",
				UseJSON:     true,
				FileEnabled: true,
				FilePath:    "./logs/app.log",
				FileSize:    100,
				MaxAge:      30,
				MaxBackups:  5,
			},
			wantErr: false,
		},
		{
			name: "minimal config",
			yamlData: `log:
  env: local
  serviceName: minimal-service`,
			want: &config.LogConfig{
				Env:         "local",
				ServiceName: "minimal-service",
				Level:       "",
				UseJSON:     false,
				FileEnabled: false,
				FilePath:    "",
				FileSize:    0,
				MaxAge:      0,
				MaxBackups:  0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpFile, err := os.CreateTemp("", "config-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.yamlData)
			require.NoError(t, err)
			require.NoError(t, tmpFile.Close())

			got, err := config.LoadFromFile(tmpFile.Name())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadFromFile_InvalidFile(t *testing.T) {
	_, err := config.LoadFromFile("nonexistent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("invalid: yaml: content:")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	_, err = config.LoadFromFile(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal config")
}

func TestLoadFromEnv(t *testing.T) {
	// Set test environment variables
	os.Setenv("BLOGGER_ENV", "test")
	os.Setenv("BLOGGER_SERVICE_NAME", "env-service")
	os.Setenv("BLOGGER_LOG_LEVEL", "debug")
	os.Setenv("BLOGGER_USE_JSON", "true")
	os.Setenv("BLOGGER_FILE_ENABLED", "true")
	os.Setenv("BLOGGER_FILE_PATH", "/tmp/test.log")
	os.Setenv("BLOGGER_FILE_SIZE", "200")
	os.Setenv("BLOGGER_MAX_AGE", "14")
	os.Setenv("BLOGGER_MAX_BACKUPS", "3")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("BLOGGER_ENV")
		os.Unsetenv("BLOGGER_SERVICE_NAME")
		os.Unsetenv("BLOGGER_LOG_LEVEL")
		os.Unsetenv("BLOGGER_USE_JSON")
		os.Unsetenv("BLOGGER_FILE_ENABLED")
		os.Unsetenv("BLOGGER_FILE_PATH")
		os.Unsetenv("BLOGGER_FILE_SIZE")
		os.Unsetenv("BLOGGER_MAX_AGE")
		os.Unsetenv("BLOGGER_MAX_BACKUPS")
	}()

	got := config.LoadFromEnv()
	expected := &config.LogConfig{
		Env:         "test",
		ServiceName: "env-service",
		Level:       "debug",
		UseJSON:     true,
		FileEnabled: true,
		FilePath:    "/tmp/test.log",
		FileSize:    200,
		MaxAge:      14,
		MaxBackups:  3,
	}

	assert.Equal(t, expected, got)
}

func TestGetDefault(t *testing.T) {
	got := config.GetDefault("local")
	expected := &config.LogConfig{
		Env:         "local",
		ServiceName: "blogger-service",
		Level:       "debug",
		UseJSON:     false,
		FileEnabled: false,
		FilePath:    "logs/app.log",
		FileSize:    100,
		MaxAge:      30,
		MaxBackups:  3,
	}

	assert.Equal(t, expected, got)
}

func TestEnvOverrides(t *testing.T) {
	// Create a base config
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	yamlData := `log:
  env: local
  serviceName: base-service
  level: info`

	_, err = tmpFile.WriteString(yamlData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Set environment override
	os.Setenv("BLOGGER_ENV", "production")
	os.Setenv("BLOGGER_LOG_LEVEL", "error")
	defer func() {
		os.Unsetenv("BLOGGER_ENV")
		os.Unsetenv("BLOGGER_LOG_LEVEL")
	}()

	got, err := config.LoadFromFile(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "production", got.Env)
	assert.Equal(t, "error", got.Level)
	assert.Equal(t, "base-service", got.ServiceName) // Should not be overridden
}