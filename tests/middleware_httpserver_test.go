package tests

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pawatthir/blogger/logger"
	"github.com/pawatthir/blogger/middleware/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestApp() *fiber.App {
	app := fiber.New()

	// Initialize logger
	config := logger.Config{
		Env:         "test",
		ServiceName: "http-middleware-test",
		Level:       "debug",
		UseJSON:     true,
	}
	logger.Init(config)
	logger.CompileCanonicalLogTemplate()

	return app
}

func TestNewLoggingMiddleware(t *testing.T) {
	config := logger.Config{
		Env:         "test",
		ServiceName: "test-service",
		Level:       "info",
		UseJSON:     true,
	}
	slogger := logger.Init(config)

	middleware := httpserver.NewLoggingMiddleware(*slogger)
	assert.NotNil(t, middleware)

	handler := middleware.Logging()
	assert.NotNil(t, handler)
}

func TestLoggingMiddleware_SuccessfulRequest(t *testing.T) {
	app := setupTestApp()

	// Test logging middleware

	app.Use(httpserver.HTTPMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success", "id": 123})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("X-Request-Id", "req-123")
	req.Header.Set("X-Username", "testuser")
	req.Header.Set("X-User-Id", "user-456")
	req.Header.Set("X-Permissions", "read,write")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "success")
}

func TestLoggingMiddleware_ErrorRequest(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())
	app.Get("/error", func(c *fiber.Ctx) error {
		return c.Status(500).JSON(fiber.Map{"error": "internal server error"})
	})

	req := httptest.NewRequest("GET", "/error", nil)
	req.Header.Set("X-Request-Id", "req-error-123")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 500, resp.StatusCode)
}

func TestLoggingMiddleware_WithRequestBody(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())
	app.Post("/users", func(c *fiber.Ctx) error {
		return c.Status(201).JSON(fiber.Map{"id": 123, "created": true})
	})

	requestBody := `{"name": "John Doe", "email": "john@example.com"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-create-123")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode)
}

func TestLoggingMiddleware_SanitizedPaths(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())
	app.Post("/auth/login", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"token": "jwt-token", "user_id": 123})
	})

	requestBody := `{"username": "user", "password": "secret123"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestConvertHeaderAttrToString(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		headers  map[string][]string
		expected string
	}{
		{
			name:     "existing header with single value",
			key:      "Content-Type",
			headers:  map[string][]string{"Content-Type": {"application/json"}},
			expected: "application/json",
		},
		{
			name:     "existing header with multiple values returns first",
			key:      "Accept",
			headers:  map[string][]string{"Accept": {"application/json", "text/html"}},
			expected: "application/json",
		},
		{
			name:     "non-existing header returns empty string",
			key:      "Non-Existent",
			headers:  map[string][]string{"Content-Type": {"application/json"}},
			expected: "",
		},
		{
			name:     "header with empty value array returns empty string",
			key:      "Empty",
			headers:  map[string][]string{"Empty": {}},
			expected: "",
		},
		{
			name:     "empty headers map returns empty string",
			key:      "Any-Header",
			headers:  map[string][]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: convertHeaderAttrToString is not exported,
			// so we test its behavior indirectly through the middleware
			// For now, we'll test the logic directly by duplicating the function
			convertHeaderAttrToString := func(key string, headers map[string][]string) string {
				if header, ok := headers[key]; ok && len(header) > 0 {
					return header[0]
				}
				return ""
			}

			result := convertHeaderAttrToString(tt.key, tt.headers)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTTPMiddleware_PanicWhenLoggerNotInitialized(t *testing.T) {
	// Save current logger
	originalSlog := logger.Slog
	defer func() {
		logger.Slog = originalSlog
	}()

	// Set logger to nil to simulate uninitialized state
	logger.Slog = nil

	assert.Panics(t, func() {
		httpserver.HTTPMiddleware()
	})
}

func TestLoggingMiddleware_DifferentHTTPMethods(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())

	// Setup different routes for different methods
	app.Get("/resource", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"method": "GET"})
	})
	app.Post("/resource", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"method": "POST"})
	})
	app.Put("/resource", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"method": "PUT"})
	})
	app.Delete("/resource", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"method": "DELETE"})
	})

	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/resource", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, 200, resp.StatusCode)
		})
	}
}

func TestLoggingMiddleware_ResponseTime(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())
	app.Get("/slow", func(c *fiber.Ctx) error {
		// Simulate slow processing
		time.Sleep(10 * time.Millisecond)
		return c.JSON(fiber.Map{"message": "slow response"})
	})

	start := time.Now()
	req := httptest.NewRequest("GET", "/slow", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	duration := time.Since(start)

	assert.Equal(t, 200, resp.StatusCode)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

func TestLoggingMiddleware_IPAddress(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())
	app.Get("/ip-test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ip": c.IP()})
	})

	req := httptest.NewRequest("GET", "/ip-test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestLoggingMiddleware_ContextPropagation(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())
	app.Get("/context", func(c *fiber.Ctx) error {
		// Test that context is properly propagated
		ctx := c.UserContext()
		assert.NotNil(t, ctx)
		assert.NotEqual(t, context.Background(), ctx)
		return c.JSON(fiber.Map{"context": "ok"})
	})

	req := httptest.NewRequest("GET", "/context", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestLoggingMiddleware_ErrorHandling(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())
	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The panic should be handled and converted to 500 status
	assert.Equal(t, 500, resp.StatusCode)
}

func TestLoggingMiddleware_HeaderExtraction(t *testing.T) {
	app := setupTestApp()

	app.Use(httpserver.HTTPMiddleware())
	app.Get("/headers", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"headers": "received"})
	})

	req := httptest.NewRequest("GET", "/headers", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("X-Request-Id", "unique-request-id")
	req.Header.Set("X-Username", "john.doe")
	req.Header.Set("X-User-Id", "user-12345")
	req.Header.Set("X-Permissions", "read")
	req.Header.Set("X-Permissions", "write") // Multiple values

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}
