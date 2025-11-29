# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

Govnilo is a Go framework for implementing keep-alive checkers and sploits (exploits) for service testing. It provides infrastructure for testing service consistency, data persistence, and running security/resilience testing.

**Module**: `github.com/HazyCorp/govnilo`  
**Go Version**: 1.24.0

## Commands

### Building

Build the main binary (typically not needed by consumers):
```bash
go build -o bin/govnilo ./internal/cmd
```

Build the example application:
```bash
cd _example
go build -o bin/example .
```

### Running

The framework provides a CLI with three main commands:

**List all registered checkers**:
```bash
./bin/example list-checkers
```

**Run a single check** (for testing):
```bash
./bin/example check --service <service-name> --checker <checker-name> --target <target-url>
```

**Run as a service** (production mode):
```bash
./bin/example run --config ./conf.yaml
```

### Testing

Run all tests:
```bash
go test ./...
```

Run tests for a specific package:
```bash
go test ./pkg/ratelimit
```

Run tests with verbose output:
```bash
go test -v ./...
```

### Dependencies

Update dependencies:
```bash
go mod tidy
```

Vendor dependencies (if needed):
```bash
go mod vendor
```

## Architecture

### Core Concepts

**Checkers**: Keep-alive checkers that verify service consistency over time. They implement the `Checker` interface with a `Check()` method that runs service flows and creates data, which should be stored in Redis for verification.

**Registration System**: Uses `govnilo.RegisterChecker()` to register checker implementations. Registration typically happens in `init()` functions and uses Uber's fx dependency injection framework.

### Package Structure

**pkg/**: Public API for developers implementing checkers
- `pkg/govnilo/`: Main public interface (Checker interface, registration functions)
- `pkg/common/hzlog/`: Logging utilities with OpenTelemetry integration
- `pkg/common/checkersettings/`: Configuration for checker execution
- `pkg/common/statestore/`: State management utilities
- `pkg/ratelimit/`: Rate limiting implementation
- `pkg/raterunner/`: Timing and execution cycle management

**internal/**: Private framework implementation
- `internal/checkerctrl/`: Orchestrates checker execution and data persistence
- `internal/hazycheck/`: Registration system and interface definitions
- `internal/cmd/`: CLI commands (check, run, list-checkers)
- `internal/fxbuild/`: Dependency injection setup
- `internal/redisbase/`: Redis storage abstractions
- `internal/taskrunner/`: Task execution infrastructure
- `internal/metricsrv/`: Metrics server

**proto/**: Protocol buffer definitions for gRPC communication

**_example/**: Example implementation showing how to use the library
- Contains example checkers in `_example/checkers/`
- Uses `replace` directive in go.mod for local development

### Key Dependencies

- **Uber fx**: Dependency injection framework
- **Cobra**: CLI framework
- **Redis**: Data persistence (go-redis/v9)
- **OpenTelemetry**: Distributed tracing
- **Zap/slog**: Structured logging

### Checker Implementation Pattern

1. **Define entity**: Implement `govnilo.Entity` interface (GetID, Encode, Decode)
2. **Create storage**: Use `govnilo.NewRedisStorage[*Entity]()` with appropriate TTL and cleanup settings
3. **Implement Checker**: 
   - Implement `Check(ctx, target)` method to run service flows
   - Store data in Redis using the storage
   - Implement `CheckerID()` method (must work with nil receiver)
4. **Register**: Call `govnilo.RegisterChecker(NewYourChecker)` in `init()`

Example skeleton:
```go
type MyChecker struct {
    l *slog.Logger
    s govnilo.RedisStorage[*MyEntity]
}

func (c *MyChecker) Check(ctx context.Context, target string) error {
    l := govnilo.GetLogger(ctx, c.l)
    // Use l.InfoContext, l.DebugContext for automatic trace/span ID inclusion
    
    // Your service flow logic
    // Save data to storage for persistence verification
    
    return nil
}

func (c *MyChecker) CheckerID() govnilo.CheckerID {
    return govnilo.CheckerID{Service: "myservice", Name: "mychecker"}
}

func init() {
    govnilo.RegisterChecker(NewMyChecker)
}
```

### OpenTelemetry Integration

The framework automatically creates spans for all Check operations. Use `govnilo.GetLogger(ctx, logger)` to get a logger that automatically includes trace_id and span_id in all log output. Always use context-aware logging methods (InfoContext, DebugContext, etc.) to ensure trace context is included.

### Configuration

Runtime configuration is provided via YAML:
- **conf.yaml**: Main application config (ports, intervals, logging, Redis connection)
- **settings.yaml**: Service-specific checker settings (targets, rate limits, scoring)

### Redis Storage

The framework provides `govnilo.RedisStorage[T]` for entity persistence:
- Automatic TTL management
- Background cleanup of expired entries
- Type-safe generic interface
- Methods: Save, GetByID, GetRandom, GetMostRecent, Delete

### Error Handling

Use `govnilo.InternalError(err)` to mark infrastructure errors vs service errors. The framework distinguishes between:
- Service errors (problems with the tested service)
- Internal errors (problems with checker infrastructure)

### Dependency Injection

Use `govnilo.RegisterConstructor()` to register shared dependencies (HTTP clients, database connections) that multiple checkers need. Constructors receive dependencies via fx and can return (T, error).

## Development Notes

- Checkers must be concurrently safe (Check is called frequently in parallel)
- CheckerID() must work with nil receiver (used during registration)
- Always use Redis for state persistence between Check calls
- The `_example` directory demonstrates best practices
- Use the example's docker-compose.yml for local Redis setup
- Binaries are built into `*/bin/` directories (gitignored)
