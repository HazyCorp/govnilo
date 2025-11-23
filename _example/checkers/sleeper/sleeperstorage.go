package sleeper

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/HazyCorp/govnilo/pkg/govnilo"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type SleeperEntity struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

func (e *SleeperEntity) GetID() string {
	return e.Id
}

func (e *SleeperEntity) Encode() ([]byte, error) {
	return json.Marshal(e)
}

func (e *SleeperEntity) Decode(data []byte) error {
	return json.Unmarshal(data, e)
}

type SleeperStorage struct {
	govnilo.RedisStorage[*SleeperEntity]
}

func NewSleeperStorage(l *slog.Logger, r *redis.Client) (*SleeperStorage, error) {
	s, err := govnilo.NewRedisStorage[*SleeperEntity](govnilo.RedisStorageInput{
		ServiceName:      "example",
		EntityNamePlural: "sleepers",
		TTL:              time.Hour,
		CleanupInterval:  time.Second,
		CleanupBatchSize: 1000,
		Logger:           l,
		RedisClient:      r,
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create redis storage")
	}

	return &SleeperStorage{RedisStorage: s}, nil
}
