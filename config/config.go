package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type LogConfig struct {
	Env         string `yaml:"env" mapstructure:"env"`
	ServiceName string `yaml:"serviceName" mapstructure:"serviceName"`
	Level       string `yaml:"level" mapstructure:"level"`
	UseJSON     bool   `yaml:"useJsonEncoder" mapstructure:"useJsonEncoder"`
	FileEnabled bool   `yaml:"fileEnabled" mapstructure:"fileEnabled"`
	FilePath    string `yaml:"filePath" mapstructure:"filePath"`
	FileSize    int    `yaml:"fileSize" mapstructure:"fileSize"`
	MaxAge      int    `yaml:"maxAge" mapstructure:"maxAge"`
	MaxBackups  int    `yaml:"maxBackups" mapstructure:"maxBackups"`
}

type Config struct {
	Log LogConfig `yaml:"log"`
}

func LoadFromFile(configPath string) (*LogConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	applyEnvOverrides(&config.Log)
	return &config.Log, nil
}

func LoadFromEnv() *LogConfig {
	// Get defaults first
	defaultEnv := getEnvOrDefault("BLOGGER_ENV", "local")
	defaults := GetDefault(defaultEnv)
	
	config := &LogConfig{
		Env:         getEnvOrDefault("BLOGGER_ENV", defaults.Env),
		ServiceName: getEnvOrDefault("BLOGGER_SERVICE_NAME", defaults.ServiceName),
		Level:       getEnvOrDefault("BLOGGER_LOG_LEVEL", defaults.Level),
		UseJSON:     getEnvBoolOrDefault("BLOGGER_USE_JSON", defaults.UseJSON),
		FileEnabled: getEnvBoolOrDefault("BLOGGER_FILE_ENABLED", defaults.FileEnabled),
		FilePath:    getEnvOrDefault("BLOGGER_FILE_PATH", defaults.FilePath),
		FileSize:    getEnvIntOrDefault("BLOGGER_FILE_SIZE", defaults.FileSize),
		MaxAge:      getEnvIntOrDefault("BLOGGER_MAX_AGE", defaults.MaxAge),
		MaxBackups:  getEnvIntOrDefault("BLOGGER_MAX_BACKUPS", defaults.MaxBackups),
	}

	if config.Env == "local" || config.Env == "development" {
		config.UseJSON = false
		config.Level = "debug"
	}

	return config
}

func GetDefault(env string) *LogConfig {
	if env == "local" || env == "development" {
		return &LogConfig{
			Env:         env,
			ServiceName: "blogger-service",
			Level:       "debug",
			UseJSON:     false,
			FileEnabled: false,
			FilePath:    "logs/app.log",
			FileSize:    100,
			MaxAge:      30,
			MaxBackups:  3,
		}
	}

	return &LogConfig{
		Env:         env,
		ServiceName: "blogger-service",
		Level:       "info",
		UseJSON:     true,
		FileEnabled: false,
		FilePath:    "logs/app.log",
		FileSize:    100,
		MaxAge:      30,
		MaxBackups:  3,
	}
}

func applyEnvOverrides(config *LogConfig) {
	if env := os.Getenv("BLOGGER_ENV"); env != "" {
		config.Env = env
	}
	if serviceName := os.Getenv("BLOGGER_SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}
	if level := os.Getenv("BLOGGER_LOG_LEVEL"); level != "" {
		config.Level = level
	}
	if useJSON := os.Getenv("BLOGGER_USE_JSON"); useJSON != "" {
		config.UseJSON = strings.ToLower(useJSON) == "true"
	}
	if fileEnabled := os.Getenv("BLOGGER_FILE_ENABLED"); fileEnabled != "" {
		config.FileEnabled = strings.ToLower(fileEnabled) == "true"
	}
	if filePath := os.Getenv("BLOGGER_FILE_PATH"); filePath != "" {
		config.FilePath = filePath
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue := parseIntSafe(value); intValue > 0 {
			return intValue
		}
	}
	return defaultValue
}

func parseIntSafe(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return result
}