package govnilo

import (
	"context"

	"github.com/HazyCorp/govnilo/internal/cmd/cmd"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/internal/redisbase"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
)

type (
	// Checker is an interface, which must be implemented by every keep-alive checker.
	// Checkers verify service consistency and availability by running operations that create and verify data persistence.
	//
	// The Check() method is called very frequently and must be concurrently safe.
	// In most cases, Check() should create some data in the service (e.g., create a user
	// and register with provided credentials) and store any necessary state in Redis
	// for later verification.
	//
	// State persistence: Any state created during Check() that needs to be verified later
	// must be stored in Redis using the redis client, that can be provided within constructor.
	// So, to create Check() that verifies data persistence, you need to get some random data from redis
	// and check that data is present and not corrupted.
	// You may use RedisStorage (see govnilo.RedisStorage) for that.
	//
	// All checkers must return the service name via CheckerID(). This name will be used in logs, metrics and some game mechanics.
	// CheckerID() MUST work with nil receiver.
	//
	// Trace ID is provided in the context for debugging purposes.
	// Use govnilo.GetLogger(ctx, logger) which automatically includes trace
	// and span IDs in log output.
	Checker = hazycheck.Checker

	// CheckerID identifies a checker by service name and checker name.
	// Every checker checks only one service and has a name (usually a user flow,
	// e.g., "stupido_user_flow"). This ID is used in logs, metrics and some game mechanics.
	// CheckerID() MUST work with nil receiver.
	CheckerID = hazycheck.CheckerID

	// RedisStorageInput provides configuration for creating a new Redis storage instance.
	// It includes service name, entity name plural, TTL settings, cleanup intervals,
	// and other storage configuration options.
	RedisStorageInput = redisbase.StorageInput

	// Entity represents a storage entity that can be serialized and deserialized.
	// Implementations must provide methods to identify, encode, and decode the entity.
	// The ID is used as the Redis key, and Encode/Decode handle serialization.
	Entity = redisbase.Entity

	// RedisStorage is a generic interface for Redis-backed entity storage.
	// It provides methods for storing, retrieving, and managing entities with TTL support
	// and automatic cleanup of expired entries.
	//
	// T must be a pointer type that implements the Entity interface (e.g., *Speaker).
	// U is the underlying type that T points to (e.g., Speaker).
	// The storage automatically handles expiration tracking and cleanup of stale entries.
	RedisStorage[T Entity] interface {
		Save(ctx context.Context, entity T) error
		GetByID(ctx context.Context, id string) (T, error)
		GetRandom(ctx context.Context) (T, error)
		GetMostRecent(ctx context.Context) (T, error)
		Delete(ctx context.Context, id string) error
		Start(ctx context.Context) error
		Stop(ctx context.Context) error
	}
)

var (
	// Execute runs the main command-line interface for govnilo.
	// It sets up signal handling, parses command-line arguments, and executes
	// the appropriate subcommand (check, run, list, etc.).
	Execute = cmd.Execute

	// RegisterChecker registers a checker constructor with the framework.
	// The constructor must be a function that returns a single Checker implementation.
	// This function should be called in an init() function to register your checker.
	// Example:
	//   func init() {
	//       govnilo.RegisterChecker(NewMyChecker)
	//   }
	RegisterChecker = hazycheck.RegisterChecker

	// RegisterConstructor registers a dependency constructor with the framework.
	// Use this to register additional dependencies (like HTTP clients, database connections)
	// that your checkers or sploits need. The constructor will be used for dependency injection.
	// Allows to avoid creation of the same entity in each constructor.
	RegisterConstructor = hazycheck.RegisterConstructor

	// InternalError wraps an internal error to mark it as an internal system error
	// rather than a service error. This is useful for distinguishing between
	// infrastructure problems and service problems in error handling.
	InternalError = hazycheck.InternalError

	// GetLogger returns a logger enriched with context attributes and trace information.
	// It extracts OpenTelemetry trace and span IDs from the context and adds them
	// as log attributes for better observability and debugging.
	GetLogger = hzlog.GetLogger
)

// NewRedisStorage creates a new Redis storage instance for entities of type T.
// T must be a pointer type that implements the Entity interface (e.g., *Speaker).
// U is the underlying type that T points to (e.g., Speaker).
// The RedisClient must be provided in the RedisStorageInput.
func NewRedisStorage[T interface {
	Entity
	~*U
}, U any](input RedisStorageInput) (RedisStorage[T], error) {
	return redisbase.NewStorage[T](input)
}
