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
- **Trace ID Management**: Automatically generates and propagates trace IDs for debugging

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

### Trace ID Usage
```go
func (c *MyChecker) Check(ctx context.Context, target string) ([]byte, error) {
    // Trace ID is automatically included in logs via hzlog context
    c.l.DebugContext(ctx, "Starting CHECK operation")
    
    // Your checker logic here...
}

func (c *MyChecker) Get(ctx context.Context, target string, data []byte) error {
    // Trace ID is automatically included in all context-aware log calls
    c.l.DebugContext(ctx, "Starting GET operation")
    c.l.InfoContext(ctx, "Processing data", slog.String("data_size", len(data)))
    return nil
}

// Optional: Retrieve trace ID for custom use
func (c *MyChecker) SomeMethod(ctx context.Context) {
    if traceID, ok := govnilo.GetTraceID(ctx); ok {
        // Use trace ID for custom logic
        fmt.Printf("Current trace: %s\n", traceID.String())
    }
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
