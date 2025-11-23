package redisbase

import "context"

// Entity represents a storage entity that can be serialized and deserialized.
// Implementations must provide methods to identify, encode, and decode the entity.
type Entity interface {
	// ID returns the entity's unique identifier as a string.
	// This ID is used as the key for storing and retrieving the entity.
	ID() string

	// Encode serializes the entity into a byte representation.
	// The returned bytes must be in a format that Decode can successfully parse.
	// Typically, this would be JSON or another serialization format.
	// Returns an error if serialization fails.
	Encode() ([]byte, error)

	// Decode parses the provided byte data and modifies the current entity instance
	// to match the deserialized data. The data format must match what Encode produces.
	// Returns an error if deserialization fails.
	Decode(data []byte) error
}

// Storage is a generic interface for Redis-backed entity storage.
// It provides methods for storing, retrieving, and managing entities with TTL support
// and automatic cleanup of expired entries.
//
// T is the entity type that must implement the Entity interface and be a pointer.
type Storage[T interface {
	Entity
	~*U
}, U any] interface {
	// Save stores an entity in Redis. The entity is stored with TTL and tracked
	// in an expiration sorted set for efficient cleanup.
	// Returns an error if the save operation fails.
	Save(ctx context.Context, entity T) error

	// GetByID retrieves an entity by its ID. Returns govnilo.ErrNotFound if the entity is not found.
	// Returns a different error if the retrieval operation fails.
	GetByID(ctx context.Context, id string) (T, error)

	// GetRandom returns a random, non-expired entity from storage.
	// This method automatically handles stale entries by removing them and retrying.
	// Returns an error if no entities are available or if the operation fails.
	// govnilo.ErrNotFound is returned if there weren't any entities in the storage.
	GetRandom(ctx context.Context) (T, error)

	// GetMostRecent returns the most recently created entity from storage.
	// This method retrieves the entity with the highest timestamp from the expiration sorted set.
	// Returns an error if no entities are available or if the operation fails.
	// govnilo.ErrNotFound is returned if there weren't any entities in the storage.
	GetMostRecent(ctx context.Context) (T, error)

	// Delete removes an entity and its associated tracking data from Redis.
	// This includes removing the entity from the expiration sorted set.
	// Returns an error if the delete operation fails.
	Delete(ctx context.Context, id string) error

	// Start initializes the storage's background cleanup routine.
	// This should be called before using the storage to enable automatic
	// cleanup of expired entries.
	Start(ctx context.Context) error

	// Stop stops the background cleanup routine and waits for it to finish.
	// This should be called during shutdown to ensure clean termination.
	Stop(ctx context.Context) error
}
