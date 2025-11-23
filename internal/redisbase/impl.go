package redisbase

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

// StorageInput provides configuration for creating a new Storage instance.
type StorageInput struct {
	// ServiceName is the name of the service. It will be used as part of the
	// Redis key prefix in the format "{ServiceName}:{EntityNamePlural}:".
	// For example, "bc" (babyconf) or "microblog".
	ServiceName string

	// EntityNamePlural is the plural name of the entity type. It will be used
	// as part of the Redis key prefix in the format "{ServiceName}:{EntityNamePlural}:".
	// For example, "speakers", "talks", "users".
	EntityNamePlural string

	// TTL is the time-to-live for stored entities. After this duration,
	// entities are considered expired and will be cleaned up.
	// Defaults to 1 hour if not specified.
	TTL time.Duration

	// CleanupInterval is the interval at which expired entries are cleaned up.
	// Defaults to 1 second if not specified.
	CleanupInterval time.Duration

	// CleanupBatchSize is the maximum number of expired entries to clean up
	// in a single cleanup cycle. Defaults to 1,000 if not specified.
	CleanupBatchSize int64

	// Logger is the logger to use for storage operations. If not provided,
	// a no-op logger will be used.
	Logger *slog.Logger

	// RedisClient is the client which will be used to communicate with the redis
	RedisClient *redis.Client
}

// storage is the private implementation of Storage interface.
type storage[T interface {
	Entity
	~*U
}, U any] struct {
	r  *redis.Client
	l  *slog.Logger
	wg sync.WaitGroup

	storageName string
	keyPrefix   string
	expZSetKey  string
	ttl         time.Duration

	cleanupInterval  time.Duration
	cleanupBatchSize int64

	cleanupCancel context.CancelFunc
	metrics       *storageMetrics
}

type EntityPtr[E Entity] interface {
	~*E
}

// NewStorage creates a new Storage instance for the given entity type.
// It takes a Redis client and configuration via StorageInput.
//
// The constructor sets up the storage with proper key prefixes, TTL tracking,
// and prepares it for background cleanup. You must call Start() before using
// the storage, and Stop() when shutting down.
//
// Example:
//
//		stor, err := redisbase.NewStorage[*Speaker](redisClient, redisbase.StorageInput{
//			ServiceName:      "babyconf",
//			EntityNamePlural: "speakers",
//			TTL:              time.Hour,
//			CleanupInterval:  time.Second,
//			CleanupBatchSize: 1_000,
//			Logger:           logger,
//	        RedisClient:      r,
//		})
func NewStorage[T interface {
	Entity
	~*U
}, U any](input StorageInput) (Storage[T, U], error) {
	var zero T
	if reflect.TypeOf(zero).Kind() != reflect.Pointer {
		return nil, errors.Errorf("type %T MUST be pointer", zero)
	}

	if input.RedisClient == nil {
		return nil, errors.New("redis client cannot be nil")
	}
	if input.ServiceName == "" {
		return nil, errors.New("service name cannot be empty")
	}
	if input.EntityNamePlural == "" {
		return nil, errors.New("entity name plural cannot be empty")
	}

	// Build key prefix from service name and entity name plural
	keyPrefix := fmt.Sprintf("%s:%s:", input.ServiceName, input.EntityNamePlural)
	storageName := fmt.Sprintf("%s:%s", input.ServiceName, input.EntityNamePlural)

	logger := input.Logger
	if logger == nil {
		logger = hzlog.NopLogger()
	}
	logger = logger.With(
		slog.String("component", "infra:redis_storage"),
		slog.String("storage_name", storageName),
		slog.String("service_name", input.ServiceName),
		slog.String("entity_name", input.EntityNamePlural),
	)

	ttl := input.TTL
	if ttl == 0 {
		ttl = time.Hour
	}

	cleanupInterval := input.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Second
	}

	cleanupBatchSize := input.CleanupBatchSize
	if cleanupBatchSize == 0 {
		cleanupBatchSize = 1_000
	}

	s := &storage[T, U]{
		r:                input.RedisClient,
		l:                logger,
		keyPrefix:        keyPrefix,
		expZSetKey:       keyPrefix + "exp",
		storageName:      storageName,
		ttl:              ttl,
		cleanupInterval:  cleanupInterval,
		cleanupBatchSize: cleanupBatchSize,
		metrics:          newStorageMetrics(storageName),
	}

	return s, nil
}

// Save stores an entity in Redis with TTL and tracks it in expiration sorted set.
func (s *storage[T, U]) Save(ctx context.Context, entity T) error {
	start := time.Now()
	defer s.metrics.SaveDuration.UpdateDuration(start)

	id := entity.GetID()
	if id == "" {
		return errors.New("entity ID cannot be empty")
	}

	createdAt := time.Now().Unix()

	data, err := entity.Encode()
	if err != nil {
		return errors.Wrap(err, "cannot encode entity")
	}

	key := s.keyFor(id)
	pipe := s.r.TxPipeline()
	pipe.Set(ctx, key, data, s.ttl)
	pipe.ZAdd(ctx, s.expZSetKey, redis.Z{Score: float64(createdAt), Member: id})

	if _, err := pipe.Exec(ctx); err != nil {
		return errors.Wrap(err, "cannot save entity to redis")
	}

	return nil
}

// GetByID retrieves an entity by its ID. Returns govnilo.ErrNotFound if id not found in redis.
func (s *storage[T, U]) GetByID(ctx context.Context, id string) (T, error) {
	start := time.Now()
	defer s.metrics.GetByIDDuration.UpdateDuration(start)

	var zero T

	if id == "" {
		return zero, errors.New("entity ID cannot be empty")
	}

	key := s.keyFor(id)
	val, err := s.r.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return zero, hazycheck.ErrNotFound
	}
	if err != nil {
		return zero, errors.Wrap(err, "cannot get entity from redis")
	}

	entity := T(new(U))
	if err := entity.Decode([]byte(val)); err != nil {
		return zero, errors.Wrap(err, "cannot decode entity")
	}

	return entity, nil
}

// GetRandom returns a random, non-expired entity from storage.
func (s *storage[T, U]) GetRandom(ctx context.Context) (T, error) {
	start := time.Now()
	defer s.metrics.GetRandomDuration.UpdateDuration(start)

	var zero T
	const maxAttempts = 5
	attempts := 0

	defer func() {
		s.metrics.GetRandomAttempts.Update(float64(attempts))
	}()

	for range maxAttempts {
		attempts++
		idCmd := s.r.ZRandMember(ctx, s.expZSetKey, 1)
		if err := idCmd.Err(); err != nil {
			if errors.Is(err, redis.Nil) {
				return zero, hazycheck.ErrNotFound
			}
			return zero, errors.Wrap(err, "cannot get random ID from redis")
		}

		ids := idCmd.Val()
		if len(ids) == 0 {
			return zero, hazycheck.ErrNotFound
		}

		id := ids[0]
		if id == "" {
			return zero, hazycheck.ErrNotFound
		}

		key := s.keyFor(id)
		getCmd := s.r.Get(ctx, key)
		if errors.Is(getCmd.Err(), redis.Nil) {
			// Stale ID, remove and retry
			s.l.DebugContext(ctx, "found stale entry in redis, retry", slog.String("id", id))

			if err := s.Delete(ctx, id); err != nil {
				s.l.WarnContext(ctx, "cannot delete stale entity from redis", slog.String("id", id), slog.Any("error", err))
			}

			continue
		}
		if err := getCmd.Err(); err != nil {
			return zero, errors.Wrap(err, "cannot get entity from redis")
		}

		entity := T(new(U))
		if err := entity.Decode([]byte(getCmd.Val())); err != nil {
			// Corrupted data, remove and retry
			s.l.WarnContext(ctx, "cannot decode entity, removing stale entry", slog.String("id", id), slog.Any("error", err))

			if err := s.Delete(ctx, id); err != nil {
				s.l.WarnContext(ctx, "cannot delete stale entity from redis", slog.String("id", id), slog.Any("error", err))
			}

			continue
		}

		return entity, nil
	}

	return zero, hazycheck.ErrNotFound
}

// GetMostRecent returns the most recently created entity from storage.
func (s *storage[T, U]) GetMostRecent(ctx context.Context) (T, error) {
	start := time.Now()
	defer s.metrics.GetByIDDuration.UpdateDuration(start)

	var zero T

	// Get the most recent ID from the sorted set (highest score = most recent timestamp)
	ids, err := s.r.ZRevRange(ctx, s.expZSetKey, 0, 0).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, hazycheck.ErrNotFound
		}
		return zero, errors.Wrap(err, "cannot get most recent ID from redis")
	}

	if len(ids) == 0 {
		return zero, hazycheck.ErrNotFound
	}

	id := ids[0]
	if id == "" {
		return zero, hazycheck.ErrNotFound
	}

	// Retrieve the entity by ID
	return s.GetByID(ctx, id)
}

// Delete removes an entity and its tracking data from Redis.
func (s *storage[T, U]) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("entity ID cannot be empty")
	}

	pipe := s.r.TxPipeline()
	pipe.Del(ctx, s.keyFor(id))
	pipe.ZRem(ctx, s.expZSetKey, id)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "cannot delete entity from redis")
	}

	return nil
}

// Start initializes the background cleanup routine.
func (s *storage[T, U]) Start(ctx context.Context) error {
	s.startCleanup(s.cleanupInterval, s.cleanupBatchSize)
	return nil
}

// Stop stops the background cleanup routine and waits for it to finish.
func (s *storage[T, U]) Stop(ctx context.Context) error {
	s.stopCleanup()
	return nil
}

// keyFor constructs a Redis key for the given entity ID.
func (s *storage[T, U]) keyFor(id string) string {
	return s.keyPrefix + id
}

// cleanupExpired removes expired entity IDs from the membership set and expiration zset.
// It finds entities where (createdAt + TTL) <= now, which means createdAt <= (now - TTL).
func (s *storage[T, U]) cleanupExpired(ctx context.Context, max int64) error {
	// Gauge on actual cleanup batch size
	s.metrics.CleanupBatchSize.Set(float64(max))

	now := time.Now().Unix()
	ttlSeconds := int64(s.ttl.Seconds())
	expiredBefore := now - ttlSeconds
	rng := &redis.ZRangeBy{Min: "-inf", Max: fmt.Sprint(expiredBefore), Offset: 0, Count: max}

	expiredIDs, err := s.r.ZRangeByScore(ctx, s.expZSetKey, rng).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}

	// Histogram on cleanup returned batch from redis
	s.metrics.CleanupReturnedBatchSize.Update(float64(len(expiredIDs)))

	if len(expiredIDs) == 0 {
		return nil
	}

	deleteStart := time.Now()
	pipe := s.r.TxPipeline()
	members := make([]any, 0, len(expiredIDs))
	for _, id := range expiredIDs {
		members = append(members, id)
	}
	pipe.ZRem(ctx, s.expZSetKey, members...)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "cannot cleanup expired entities")
	}

	// Histogram on duration of cleanup deleting batch
	s.metrics.CleanupDeleteDuration.UpdateDuration(deleteStart)

	return nil
}

// startCleanup launches a background goroutine that periodically removes expired IDs.
func (s *storage[T, U]) startCleanup(interval time.Duration, batch int64) {
	if s.cleanupCancel != nil {
		// already started
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cleanupCancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.cleanupExpired(ctx, batch); err != nil {
					s.l.WarnContext(ctx, "cannot cleanup expired entities", slog.Any("error", err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// stopCleanup stops the background cleanup goroutine if running.
func (s *storage[T, U]) stopCleanup() {
	if s.cleanupCancel != nil {
		s.cleanupCancel()
		s.cleanupCancel = nil

		s.wg.Wait()
	}
}
