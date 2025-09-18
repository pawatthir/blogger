# Blogger - Production-Ready Go Logging Library

A comprehensive, production-ready logging library for Go applications that combines the power of `slog` (structured logging) with `Zap` (high-performance) and `OpenTelemetry` for distributed tracing.

## Features

- 🚀 **High Performance**: Built on Zap for minimal overhead
- 📊 **Structured Logging**: Uses Go's standard `log/slog` interface
- 🔗 **Distributed Tracing**: OpenTelemetry integration with trace/span IDs
- 🐕 **Datadog Ready**: Built-in DD_ENV, DD_SERVICE, DD_VERSION support
- 🔒 **Security**: Automatic sensitive data sanitization
- 🌐 **Middleware**: Ready-to-use HTTP and gRPC interceptors
- ⚙️ **Configurable**: Environment-based configuration
- 📝 **Canonical Logging**: Consistent log format across services

## Quick Start

### Installation

```bash
go get github.com/your-username/blogger
```

### Basic Usage

```go
package main

import (
    "context"
    "log/slog"

    "github.com/your-username/blogger/logger"
)

func main() {
    // Initialize logger
    log := logger.Init(logger.Config{
        Env:         "production",
        ServiceName: "order-service",
        Level:       "info",
        UseJSON:     true,
    })

    // Use structured logging
    log.InfoContext(context.Background(), "service started",
        slog.String("port", "8080"),
        slog.String("version", "v1.0.0"))
}
```

### Configuration Options

#### Environment Variables

```bash
# Datadog Integration (automatically detected)
export DD_ENV="production"
export DD_SERVICE="order-service"
export DD_VERSION="v1.2.3"

# Blogger Configuration
export BLOGGER_ENV="production"
export BLOGGER_SERVICE_NAME="order-service"
export BLOGGER_LOG_LEVEL="info"
export BLOGGER_USE_JSON="true"
export BLOGGER_FILE_ENABLED="false"
```

#### Programmatic Configuration

```go
config := logger.Config{
    Env:         "production",     // Environment: local, development, production
    ServiceName: "order-service",  // Service name for logs
    Level:       "info",          // Log level: debug, info, warn, error
    UseJSON:     true,            // JSON format for production
    FileEnabled: false,           // Enable file logging
    FilePath:    "logs/app.log",  // Log file path
    FileSize:    100,             // Max file size in MB
    MaxAge:      30,              // Max age in days
    MaxBackups:  3,               // Max backup files
}

logger.Init(config)
```

#### YAML Configuration

```yaml
# config.yaml
log:
  env: production
  serviceName: order-service
  level: info
  useJsonEncoder: true
  fileEnabled: false
  filePath: logs/app.log
  fileSize: 100
  maxAge: 30
  maxBackups: 3
```

```go
// Load from YAML
config, err := config.LoadFromFile("config.yaml")
if err != nil {
    panic(err)
}
logger.Init(*config)
```

## Middleware Integration

### HTTP Server (Fiber)

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/your-username/blogger/logger"
    "github.com/your-username/blogger/middleware/httpserver"
)

func main() {
    // Initialize logger
    logger.Init(logger.Config{
        Env:         "production",
        ServiceName: "api-gateway",
    })

    // Create Fiber app
    app := fiber.New()

    // Add logging middleware
    app.Use(httpserver.HTTPMiddleware())

    app.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{"status": "ok"})
    })

    app.Listen(":8080")
}
```

### gRPC Server

```go
package main

import (
    "google.golang.org/grpc"
    "github.com/your-username/blogger/logger"
    "github.com/your-username/blogger/middleware/grpcserver"
)

func main() {
    // Initialize logger
    logger.Init(logger.Config{
        Env:         "production",
        ServiceName: "user-service",
    })

    // Create gRPC server with logging interceptor
    server := grpc.NewServer(
        grpc.UnaryInterceptor(
            grpcserver.GRPCServerInterceptor(),
        ),
    )

    // Register your services...
    // pb.RegisterUserServiceServer(server, &userService{})

    // Start server...
}
```

### gRPC Client

```go
package main

import (
    "google.golang.org/grpc"
    "github.com/your-username/blogger/middleware/grpcclient"
)

func main() {
    // Create gRPC client with logging interceptor
    conn, err := grpc.Dial("localhost:50051",
        grpc.WithInsecure(),
        grpc.WithUnaryInterceptor(
            grpcclient.GRPCClientInterceptor(),
        ),
    )
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // Use your client...
}
```

## Advanced Usage

### Service-Level Logging

```go
type OrderService struct {
    logger logger.Pathfinder
}

func NewOrderService() *OrderService {
    return &OrderService{
        logger: logger.NewPathfinder("order"),
    }
}

func (s *OrderService) CreateOrder(ctx context.Context, order Order) error {
    s.logger.InfoContext(ctx, "creating order",
        slog.String("order_id", order.ID),
        slog.Int("items_count", len(order.Items)))

    // Business logic...

    if err != nil {
        s.logger.ErrorContext(ctx, "failed to create order",
            slog.String("error", err.Error()))
        return err
    }

    s.logger.InfoContext(ctx, "order created successfully")
    return nil
}
```

### PostgreSQL Integration

```go
package main

import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v5/tracelog"
    "github.com/your-username/blogger/logger"
)

func main() {
    logger.Init(logger.Config{
        Env:         "production",
        ServiceName: "database-service",
    })

    config, _ := pgxpool.ParseConfig("postgres://user:pass@localhost/db")

    // Add logger to PostgreSQL
    config.ConnConfig.Tracer = &tracelog.TraceLog{
        Logger:   logger.NewPGXLoggerFromSlog(),
        LogLevel: tracelog.LogLevelInfo,
    }

    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    // Use pool...
}
```

### Custom Fields and Context

```go
// Add custom fields to all logs
customLogger := slog.Default().With(
    slog.String("region", "us-west-2"),
    slog.String("cluster", "prod-1"),
)

// Use context-aware logging
func processRequest(ctx context.Context, req Request) error {
    // Trace ID and span ID automatically added from OpenTelemetry context
    slog.InfoContext(ctx, "processing request",
        slog.String("request_id", req.ID),
        slog.Group("user",
            slog.String("id", req.UserID),
            slog.String("email", req.UserEmail),
        ),
    )

    return nil
}
```

### Runtime Log Level Changes

```go
// Get current logger
currentLogger := logger.Slog

// Create new logger with different level
newConfig := logger.Config{
    Env:         "production",
    ServiceName: "dynamic-service",
    Level:       "debug", // Changed from info to debug
}

logger.Init(newConfig)
```

## Log Output Examples

### Development/Local Environment

```
2024-01-15T10:30:45Z	INFO	[http][internal] POST 200 /api/orders 125ms - Order created	{"request": {"user_id": "123", "items": [...]}, "response": {"order_id": "ord_456"}}
```

### Production Environment (JSON)

```json
{
  "level": "INFO",
  "time": "2024-01-15T10:30:45Z",
  "msg": "[http][internal] POST 200 /api/orders 125ms - Order created",
  "trace_id": "abc123def456",
  "span_id": "789xyz",
  "dd": {
    "env": "production",
    "service": "order-service",
    "version": "v1.2.3",
    "trace_id": "abc123def456",
    "span_id": "789xyz"
  },
  "httpserver_md": {
    "type": "httpserver",
    "method": "POST",
    "path": "/api/orders",
    "duration": "125ms",
    "ip": "192.168.1.100",
    "x-request-id": "req-789"
  },
  "request": {
    "user_id": "123",
    "items": [{"id": "item1", "quantity": 2}]
  },
  "response": {
    "order_id": "ord_456",
    "status": "created"
  }
}
```

## Security Features

### Automatic Data Sanitization

The library automatically redacts sensitive information from logs:

```go
// These paths are automatically sanitized:
var DenyPatterns = []string{
    "login",
    "refresh-token",
    "verify-otp",
    "password",
    "token",
    "secret",
    "key",
}

// Sensitive fields in structs
type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password" sensitive:"true"` // Automatically masked
}
```

### Custom Sanitization

```go
// Add custom deny patterns
logger.DenyPatterns = append(logger.DenyPatterns, "api-key", "session")
```

## Environment Defaults

### Local/Development
- **Format**: Pretty console output with colors
- **Level**: Debug
- **JSON**: Disabled
- **File logging**: Disabled

### Production
- **Format**: JSON structured logs
- **Level**: Info
- **JSON**: Enabled
- **Datadog fields**: Included
- **Trace correlation**: Enabled

## Best Practices

1. **Always use context**: Pass context to logging methods for trace correlation
2. **Structured fields**: Use `slog.String()`, `slog.Int()` etc. instead of string formatting
3. **Service loggers**: Create service-specific loggers with `NewPathfinder()`
4. **Error logging**: Always log errors with context and relevant fields
5. **Performance**: Use lazy evaluation for expensive debug operations
6. **Security**: Never log sensitive data directly

```go
// ✅ Good
slog.InfoContext(ctx, "user created",
    slog.String("user_id", userID),
    slog.String("email", email))

// ❌ Bad
slog.InfoContext(ctx, fmt.Sprintf("user %s created with email %s", userID, email))
```

## Troubleshooting

### Logger Not Initialized
```go
if logger.Slog == nil {
    panic("Logger not initialized. Call logger.Init() first.")
}
```

### Missing Trace IDs
Ensure OpenTelemetry is properly configured in your application context.

### Performance Issues
- Check log level configuration
- Avoid logging in hot paths
- Use conditional logging for debug statements

### Missing Datadog Fields
Set environment variables:
```bash
export DD_ENV="production"
export DD_SERVICE="your-service"
export DD_VERSION="v1.0.0"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## License

MIT License - see LICENSE file for details.

---

**Built with ❤️ for production Go applications**