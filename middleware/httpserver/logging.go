package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pawatthir/blogger/logger"
)

func convertHeaderAttrToString(key string, headers map[string][]string) string {
	if header, ok := headers[key]; ok && len(header) > 0 {
		return header[0]
	}
	return ""
}

type LoggingMiddleware interface {
	Logging() fiber.Handler
}

type loggingMiddleware struct {
	logger slog.Logger
}

func NewLoggingMiddleware(slogger slog.Logger) LoggingMiddleware {
	loggerWithName := slogger.With(slog.String("logger_name", "http_middleware"))
	return &loggingMiddleware{
		logger: *loggerWithName,
	}
}

func (l *loggingMiddleware) Logging() fiber.Handler {
	return func(c *fiber.Ctx) error {
		startTime := time.Now()
		requestBody := c.Body()

		// Set up a custom context for the request
		ctx := context.WithValue(c.UserContext(), "middleware", "http")
		c.SetUserContext(ctx)

		// Add panic recovery
		defer func() {
			if r := recover(); r != nil {
				c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
			}
		}()

		err := c.Next()
		elapse := time.Since(startTime)
		responseBody := c.Response().Body()
		headers := c.GetReqHeaders()

		var fields []any
		fields = append(fields,
			slog.String("logger_name", "canonical"),
			slog.Group("httpserver_md",
				slog.String("type", "httpserver"),
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("ip", c.IP()),
				slog.String("duration", elapse.String()),
				slog.String("accept-language", convertHeaderAttrToString("Accept-Language", headers)),
				slog.String("x-request-id", convertHeaderAttrToString("X-Request-Id", headers)),
				slog.String("x-username", convertHeaderAttrToString("X-Username", headers)),
				slog.String("x-user-id", convertHeaderAttrToString("X-User-Id", headers)),
				slog.String("x-permissions", fmt.Sprint(headers["X-Permissions"])),
			),
		)

		var level logger.Level
		if c.Response().StatusCode() >= http.StatusBadRequest {
			level = logger.Error
		} else {
			level = logger.Info
		}

		logger.CanonicalLogger(
			c.UserContext(),
			l.logger,
			level,
			requestBody,
			responseBody,
			err,
			logger.CanonicalLog{
				Transport: "http",
				Traffic:   "internal",
				Method:    c.Method(),
				Status:    c.Response().StatusCode(),
				Path:      c.Path(),
				Duration:  elapse,
			},
			fields,
		)
		return err
	}
}

func HTTPMiddleware() fiber.Handler {
	if logger.Slog == nil {
		panic("Logger not initialized. Call logger.Init() first.")
	}
	return NewLoggingMiddleware(*logger.Slog).Logging()
}
