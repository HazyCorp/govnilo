package hzlog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"
)

// TraceIDKey is the context key for trace ID
type TraceIDKey struct{}

// TraceID represents a unique identifier for tracing operations
type TraceID string

// String returns the string representation of the trace ID
func (t TraceID) String() string {
	return string(t)
}

// GenerateTraceID generates a new random trace ID
func GenerateTraceID() TraceID {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return TraceID(fmt.Sprintf("%d", time.Now().UnixNano()))
	}
	return TraceID(hex.EncodeToString(bytes))
}

// WithTraceID adds a trace ID to the context and hzlog context
func WithTraceID(ctx context.Context, traceID TraceID) context.Context {
	// Add trace ID to the context for retrieval
	ctx = context.WithValue(ctx, TraceIDKey{}, traceID)
	// Add trace ID to hzlog context for automatic logging
	ctx = ContextWith(ctx, slog.String("trace_id", traceID.String()))
	return ctx
}

// GetTraceID retrieves the trace ID from the context
func GetTraceID(ctx context.Context) (TraceID, bool) {
	traceID, ok := ctx.Value(TraceIDKey{}).(TraceID)
	return traceID, ok
}

// MustGetTraceID retrieves the trace ID from the context, panics if not found
func MustGetTraceID(ctx context.Context) TraceID {
	traceID, ok := GetTraceID(ctx)
	if !ok {
		panic("trace ID not found in context")
	}
	return traceID
}

// WithNewTraceID creates a new context with a generated trace ID
func WithNewTraceID(ctx context.Context) context.Context {
	return WithTraceID(ctx, GenerateTraceID())
}

// LogWithTraceID creates a logger with trace ID context using hzlog
func LogWithTraceID(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if traceID, ok := GetTraceID(ctx); ok {
		// Use hzlog's context system to add trace_id to all log messages
		ctxWithTrace := ContextWith(ctx, slog.String("trace_id", traceID.String()))
		return GetLogger(ctxWithTrace, logger)
	}
	return logger
}

// FormatTraceID formats a trace ID for logging
func FormatTraceID(traceID TraceID) string {
	return fmt.Sprintf("[%s]", traceID.String())
}
