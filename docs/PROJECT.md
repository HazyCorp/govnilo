# Govnilo Project Documentation

## Overview
Govnilo is a Go framework for implementing keep-alive checkers and sploits (exploits) for services. It provides infrastructure for testing service consistency and running attacks against target services.

## Core Concepts

### Keep-Alive Checkers
Keep-alive checkers verify that services maintain consistency over time. They follow a CHECK/GET pattern:

#### CHECK Operation
- Runs the most common flow of your service
- Creates some data that should persist (e.g., user credentials, session data)
- Returns this data as `[]byte` for later verification
- Called frequently to test service functionality

#### GET Operation
- Verifies that data from CHECK still exists in the service
- Tests service consistency over time
- Ensures the service hasn't dropped or corrupted the data
- Called less frequently than CHECK, only for randomly chosen CHECK calls

### Sploits (Exploits)
Sploits run attacks against target services to test their resilience:

#### RunAttack Operation
- Executes an attack on the service at the target
- Tests how the service handles malicious or problematic requests
- Used for security testing and resilience validation

## Architecture

### Registration System
- **hazycheck.RegisterChecker()**: Registers checker implementations
- **hazycheck.RegisterSploit()**: Registers sploit implementations
- Uses dependency injection (fx) for constructor parameters
- Registration typically happens in `init()` functions

### Core Components
- **checkerctrl/**: Orchestrates checker execution and data persistence
- **hazycheck/**: Registration system and interface definitions
- **common/**: Shared utilities for logging, state management
- **ratelimit/**: Controls execution rate of operations
- **raterunner/**: Manages timing and execution cycles

### Infrastructure Features
- **Data Persistence**: Automatically saves data between CHECK and GET operations
- **Rate Limiting**: Controlled execution with configurable timing
- **State Management**: Handles cleanup of old/stale data
- **Concurrent Safety**: Checkers must be safe for concurrent execution
- **Error Handling**: Comprehensive error handling and context cancellation
- **OpenTelemetry Tracing**: Automatically generates and propagates OpenTelemetry spans for debugging and observability

## Developer Experience

### Implementing Checkers
```go
type MyChecker struct {
    l *slog.Logger
    c *http.Client
}

func (c *MyChecker) Check(ctx context.Context, target string) ([]byte, error) {
    // Run service flow, create data
    // Return data that should persist
}

func (c *MyChecker) Get(ctx context.Context, target string, data []byte) error {
    // Verify data still exists in service
}

func (c *MyChecker) CheckerID() CheckerID {
    return CheckerID{Service: "myservice", Name: "mychecker"}
}

func init() {
    hazycheck.RegisterChecker(NewMyChecker)
}
```

### Implementing Sploits
```go
type MySploit struct {
    l *slog.Logger
}

func (s *MySploit) RunAttack(ctx context.Context, target string) error {
    // Execute attack against target service
}

func (s *MySploit) SploitID() SploitID {
    return SploitID{Service: "myservice", Name: "myattack"}
}

func init() {
    hazycheck.RegisterSploit(NewMySploit)
}
```

### OpenTelemetry Tracing Usage
```go
import (
    "github.com/HazyCorp/govnilo/pkg/govnilo"
)

func (c *MyChecker) Check(ctx context.Context, target string) ([]byte, error) {
    // Use govnilo.GetLogger to get a logger with automatic trace/span ID inclusion
    logger := govnilo.GetLogger(ctx, c.l)
    
    // Trace ID and span ID are automatically included in logs via hzlog context
    logger.DebugContext(ctx, "Starting CHECK operation")
    
    // Your checker logic here...
    
    logger.InfoContext(ctx, "Check operation completed")
    
    return data, nil
}

func (c *MyChecker) Get(ctx context.Context, target string, data []byte) error {
    // Use govnilo.GetLogger to get a logger with automatic trace/span ID inclusion
    logger := govnilo.GetLogger(ctx, c.l)
    
    // Trace ID and span ID are automatically included in all context-aware log calls
    logger.DebugContext(ctx, "Starting GET operation")
    logger.InfoContext(ctx, "Processing data", slog.String("data_size", len(data)))
    
    logger.InfoContext(ctx, "Data verification completed")
    
    return nil
}
```

## OpenTelemetry Integration

Govnilo uses OpenTelemetry for distributed tracing and observability. The framework automatically creates spans for Check, Get, and RunAttack operations, and trace/span IDs are automatically included in all logs.

### Automatic Span Creation

The framework automatically creates OpenTelemetry spans for:
- **Checker operations**: `checker.Check` and `checker.Get` 
- **Sploit operations**: `sploit.RunAttack`

### Logging with Trace Context

Use `govnilo.GetLogger` to get a logger that automatically includes trace and span IDs:

```go
func (c *MyChecker) Check(ctx context.Context, target string) ([]byte, error) {
    // Get logger with automatic trace/span ID inclusion
    logger := govnilo.GetLogger(ctx, c.l)
    
    // All logs will automatically include trace_id and span_id
    logger.InfoContext(ctx, "Starting check operation", 
        slog.String("target", target))
    
    // Your checker logic here...
    data, err := c.performCheck(target)
    if err != nil {
        logger.ErrorContext(ctx, "Check operation failed", 
            slog.String("error", err.Error()))
        return nil, err
    }
    
    logger.InfoContext(ctx, "Check operation completed", 
        slog.Int("data_size", len(data)))
    
    return data, nil
}
```

### Accessing Trace Context

The recommended approach is to use `govnilo.GetLogger` which automatically includes trace and span IDs in all logs:

```go
func (c *MyChecker) SomeMethod(ctx context.Context) {
    // Use govnilo.GetLogger for automatic trace/span ID inclusion
    logger := govnilo.GetLogger(ctx, c.l)
    
    // All logs will automatically include trace_id and span_id
    logger.InfoContext(ctx, "Processing with trace context")
    
    // If you need direct access to trace/span IDs, you can still use OpenTelemetry:
    // import "go.opentelemetry.io/otel/trace"
    // spanCtx := trace.SpanContextFromContext(ctx)
    // traceID := spanCtx.TraceID().String()
}
```

### Log Output

With OpenTelemetry integration, your logs will include trace and span IDs:

```json
{
  "level": "info",
  "msg": "Check operation completed",
  "trace_id": "28d17438dca4eeb82bdd1c98f2ac2b0a",
  "span_id": "ad8eb41196eb9978",
  "target": "example.com"
}
```

## Use Cases

### Service Monitoring
- Continuous health checking of services
- Data consistency validation
- Performance monitoring over time

### Security Testing
- Attack simulation and testing
- Resilience validation
- Vulnerability assessment

### Quality Assurance
- Automated testing of service behavior
- Regression testing for data persistence
- Load testing with rate limiting

## Implementation Examples

See the `_example/` directory for concrete examples of:
- Checker implementations with CHECK/GET pattern
- Sploit implementations for attack simulation
- Proper registration and dependency injection
- Error handling and context usage
