# Blogger Library - User Guide

## Overview

Blogger is a production-ready Go logging library that combines the power of `slog` (structured logging) with `Zap` (high-performance) and `OpenTelemetry` for distributed tracing. This guide will help you get started with using the Blogger library in your Go applications.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Configuration](#configuration)
4. [Basic Logging](#basic-logging)
5. [Middleware Integration](#middleware-integration)
6. [Advanced Features](#advanced-features)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

## Installation

Add the Blogger library to your Go project:

```bash
go get github.com/pawatthir/blogger
```

## Quick Start

### 1. Initialize the Logger

```go
package main

import (
    "context"
    "log/slog"
    "github.com/pawatthir/blogger/logger"
)

func main() {
    // Initialize logger with basic configuration
    log := logger.Init(logger.Config{
        Env:         "development",
        ServiceName: "my-service",
        Level:       "info",
        UseJSON:     false, // Pretty console output for development
    })

    // Log a simple message
    log.InfoContext(context.Background(), "Application started successfully")
}
```

### 2. Using Structured Logging

```go
// Log with structured fields
slog.InfoContext(ctx, "User logged in",
    slog.String("user_id", "12345"),
    slog.String("email", "user@example.com"),
    slog.Duration("login_time", time.Since(start)))

// Group related fields
slog.InfoContext(ctx, "Order processed",
    slog.Group("order",
        slog.String("id", "ord_123"),
        slog.Int("items", 3),
        slog.Float64("total", 99.99)),
    slog.Group("customer",
        slog.String("id", "cust_456"),
        slog.String("email", "customer@example.com")))
```

## Configuration

### Environment Variables

Set these environment variables to configure the logger:

```bash
# Datadog Integration
export DD_ENV="production"
export DD_SERVICE="my-service"
export DD_VERSION="v1.0.0"

# Blogger Configuration
export BLOGGER_ENV="production"
export BLOGGER_SERVICE_NAME="my-service"
export BLOGGER_LOG_LEVEL="info"
export BLOGGER_USE_JSON="true"
export BLOGGER_FILE_ENABLED="false"
```

### Programmatic Configuration

```go
config := logger.Config{
    Env:         "production",     // Environment: local, development, production
    ServiceName: "my-service",     // Service name for logs
    Level:       "info",          // Log level: debug, info, warn, error
    UseJSON:     true,            // JSON format for production
    FileEnabled: true,            // Enable file logging
    FilePath:    "logs/app.log",  // Log file path
    FileSize:    100,             // Max file size in MB
    MaxAge:      30,              // Max age in days
    MaxBackups:  3,               // Max backup files
}

logger.Init(config)
```

### YAML Configuration

Create a `config.yaml` file:

```yaml
log:
  env: production
  serviceName: my-service
  level: info
  useJsonEncoder: true
  fileEnabled: true
  filePath: logs/app.log
  fileSize: 100
  maxAge: 30
  maxBackups: 3
```

Load configuration from YAML:

```go
config, err := config.LoadFromFile("config.yaml")
if err != nil {
    panic(err)
}
logger.Init(*config)
```

## Basic Logging

### Log Levels

```go
// Debug - for detailed information during development
slog.DebugContext(ctx, "Processing user request", slog.String("user_id", userID))

// Info - for general information about application flow
slog.InfoContext(ctx, "User authentication successful")

// Warn - for potentially harmful situations
slog.WarnContext(ctx, "High memory usage detected", slog.Int("memory_mb", 850))

// Error - for error conditions
slog.ErrorContext(ctx, "Database connection failed", slog.String("error", err.Error()))
```

### Error Logging

```go
func processOrder(ctx context.Context, orderID string) error {
    err := validateOrder(orderID)
    if err != nil {
        slog.ErrorContext(ctx, "Order validation failed",
            slog.String("order_id", orderID),
            slog.String("error", err.Error()),
            slog.String("operation", "validate_order"))
        return err
    }
    
    slog.InfoContext(ctx, "Order processed successfully",
        slog.String("order_id", orderID))
    return nil
}
```

## Middleware Integration

### HTTP Server (Fiber)

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/pawatthir/blogger/logger"
    "github.com/pawatthir/blogger/middleware/httpserver"
)

func main() {
    // Initialize logger
    logger.Init(logger.Config{
        Env:         "production",
        ServiceName: "api-service",
    })

    // Create Fiber app
    app := fiber.New()

    // Add logging middleware
    app.Use(httpserver.HTTPMiddleware())

    // Your routes
    app.Get("/users/:id", getUserHandler)
    app.Post("/orders", createOrderHandler)

    app.Listen(":8080")
}

func getUserHandler(c *fiber.Ctx) error {
    userID := c.Params("id")
    
    // The middleware automatically logs request/response details
    // You can add business logic logging
    slog.InfoContext(c.Context(), "Fetching user details",
        slog.String("user_id", userID))
    
    return c.JSON(fiber.Map{"user_id": userID, "name": "John Doe"})
}
```

### gRPC Server

```go
package main

import (
    "google.golang.org/grpc"
    "github.com/pawatthir/blogger/logger"
    "github.com/pawatthir/blogger/middleware/grpcserver"
)

func main() {
    // Initialize logger
    logger.Init(logger.Config{
        Env:         "production",
        ServiceName: "grpc-service",
    })

    // Create gRPC server with logging interceptor
    server := grpc.NewServer(
        grpc.UnaryInterceptor(grpcserver.GRPCServerInterceptor()),
    )

    // Register your services
    // pb.RegisterUserServiceServer(server, &userService{})

    // Start server...
}
```

### gRPC Client

```go
package main

import (
    "google.golang.org/grpc"
    "github.com/pawatthir/blogger/middleware/grpcclient"
)

func main() {
    // Create gRPC client with logging interceptor
    conn, err := grpc.Dial("localhost:50051",
        grpc.WithInsecure(),
        grpc.WithUnaryInterceptor(grpcclient.GRPCClientInterceptor()),
    )
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // The interceptor will automatically log all gRPC calls
}
```

## Advanced Features

### Service-Level Loggers

Create dedicated loggers for different parts of your application:

```go
type UserService struct {
    logger logger.Pathfinder
}

func NewUserService() *UserService {
    return &UserService{
        logger: logger.NewPathfinder("user-service"),
    }
}

func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) error {
    s.logger.InfoContext(ctx, "Creating new user",
        slog.String("email", req.Email),
        slog.String("role", req.Role))

    // Business logic...

    if err != nil {
        s.logger.ErrorContext(ctx, "Failed to create user",
            slog.String("error", err.Error()),
            slog.String("email", req.Email))
        return err
    }

    s.logger.InfoContext(ctx, "User created successfully",
        slog.String("user_id", userID))
    return nil
}
```

### Database Integration (PostgreSQL)

```go
package main

import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v5/tracelog"
    "github.com/pawatthir/blogger/logger"
)

func setupDatabase() *pgxpool.Pool {
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
    if err != nil {
        panic(err)
    }
    
    return pool
}
```

### Custom Context Fields

```go
// Add custom fields to all subsequent logs
customLogger := slog.Default().With(
    slog.String("region", "us-west-2"),
    slog.String("cluster", "prod-1"),
    slog.String("version", "v1.2.3"))

// Use context to pass request-specific data
func handleRequest(ctx context.Context, requestID string) {
    // Add request ID to context for all logs in this function
    ctx = context.WithValue(ctx, "request_id", requestID)
    
    customLogger.InfoContext(ctx, "Processing request")
    
    // All subsequent logs will include the custom fields
}
```

## Best Practices

### 1. Always Use Context

```go
// ✅ Good - includes trace correlation
slog.InfoContext(ctx, "User authenticated", slog.String("user_id", userID))

// ❌ Bad - missing context
slog.Info("User authenticated", slog.String("user_id", userID))
```

### 2. Use Structured Fields

```go
// ✅ Good - structured and searchable
slog.InfoContext(ctx, "Order created",
    slog.String("order_id", orderID),
    slog.Float64("amount", 99.99),
    slog.Int("items", 3))

// ❌ Bad - string formatting
slog.InfoContext(ctx, fmt.Sprintf("Order %s created with amount %.2f and %d items", 
    orderID, 99.99, 3))
```

### 3. Group Related Fields

```go
// ✅ Good - organized groups
slog.InfoContext(ctx, "Payment processed",
    slog.Group("payment",
        slog.String("id", paymentID),
        slog.String("method", "credit_card"),
        slog.Float64("amount", 99.99)),
    slog.Group("customer",
        slog.String("id", customerID),
        slog.String("email", "customer@example.com")))
```

### 4. Handle Errors Properly

```go
func processPayment(ctx context.Context, paymentID string) error {
    slog.InfoContext(ctx, "Processing payment", slog.String("payment_id", paymentID))
    
    err := chargeCard(paymentID)
    if err != nil {
        slog.ErrorContext(ctx, "Payment processing failed",
            slog.String("payment_id", paymentID),
            slog.String("error", err.Error()),
            slog.String("operation", "charge_card"))
        return fmt.Errorf("payment failed: %w", err)
    }
    
    slog.InfoContext(ctx, "Payment processed successfully",
        slog.String("payment_id", paymentID))
    return nil
}
```

### 5. Use Appropriate Log Levels

- **Debug**: Detailed information for debugging (disabled in production)
- **Info**: General application flow and business events
- **Warn**: Potentially harmful situations that don't break functionality
- **Error**: Error conditions that affect functionality

## Troubleshooting

### Common Issues

#### 1. Logger Not Initialized

```go
// Always initialize before using
if logger.Slog == nil {
    panic("Logger not initialized. Call logger.Init() first.")
}
```

#### 2. Missing Trace IDs

Ensure OpenTelemetry is properly configured:

```go
// Make sure your context contains tracing information
import "go.opentelemetry.io/otel/trace"

span := trace.SpanFromContext(ctx)
if span.SpanContext().IsValid() {
    // Tracing is working
}
```

#### 3. Performance Issues

```go
// Use conditional logging for expensive operations
if slog.Default().Enabled(ctx, slog.LevelDebug) {
    expensiveData := generateExpensiveDebugData()
    slog.DebugContext(ctx, "Debug info", slog.Any("data", expensiveData))
}
```

#### 4. Missing Datadog Fields

Set required environment variables:

```bash
export DD_ENV="production"
export DD_SERVICE="your-service"
export DD_VERSION="v1.0.0"
```

### Log Output Examples

#### Development Environment

```
2024-01-15T10:30:45Z	INFO	Processing order	{"order_id": "ord_123", "customer_id": "cust_456"}
```

#### Production Environment (JSON)

```json
{
  "level": "INFO",
  "time": "2024-01-15T10:30:45Z",
  "msg": "Processing order",
  "trace_id": "abc123def456",
  "span_id": "789xyz",
  "dd": {
    "env": "production",
    "service": "order-service",
    "version": "v1.2.3"
  },
  "order_id": "ord_123",
  "customer_id": "cust_456"
}
```

## Security Considerations

The library automatically sanitizes sensitive information. These fields/paths are redacted by default:

- login
- refresh-token
- verify-otp
- password
- token
- secret
- key

You can add custom patterns:

```go
// Add custom deny patterns
logger.DenyPatterns = append(logger.DenyPatterns, "api-key", "session")
```

## Support

For issues, questions, or contributions, please visit the project repository or contact the development team.

---

*This user guide covers the essential features of the Blogger library. For more advanced use cases and API documentation, refer to the source code and additional documentation.*