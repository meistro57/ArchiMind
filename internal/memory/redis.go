// internal/memory/redis.go
package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"archimind/internal/config"

	"github.com/redis/go-redis/v9"
)

type RedisMemory struct {
	client *redis.Client
	ttl    time.Duration
}

type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewRedisMemory(cfg config.Config) *RedisMemory {
	return &RedisMemory{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}),
		ttl: time.Duration(cfg.RedisTTLSeconds) * time.Second,
	}
}

func (m *RedisMemory) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return m.client.Ping(ctx).Err()
}

func (m *RedisMemory) Close() error {
	return m.client.Close()
}

func (m *RedisMemory) SaveTurn(ctx context.Context, sessionID string, role string, content string, maxTurns int) error {
	key := fmt.Sprintf("chat:%s:history", sessionID)

	turn := Turn{
		Role:    role,
		Content: content,
	}

	raw, err := json.Marshal(turn)
	if err != nil {
		return err
	}

	if err := m.client.RPush(ctx, key, raw).Err(); err != nil {
		return err
	}

	if maxTurns > 0 {
		if err := m.client.LTrim(ctx, key, int64(-maxTurns), -1).Err(); err != nil {
			return err
		}
	}

	return m.client.Expire(ctx, key, 24*time.Hour).Err()
}

func (m *RedisMemory) GetHistory(ctx context.Context, sessionID string) ([]Turn, error) {
	key := fmt.Sprintf("chat:%s:history", sessionID)

	items, err := m.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	turns := make([]Turn, 0, len(items))

	for _, item := range items {
		var turn Turn
		if err := json.Unmarshal([]byte(item), &turn); err == nil {
			turns = append(turns, turn)
		}
	}

	return turns, nil
}

func (m *RedisMemory) GetJSON(ctx context.Context, key string, target any) (bool, error) {
	raw, err := m.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return false, err
	}

	return true, nil
}

func (m *RedisMemory) SetJSON(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return m.client.Set(ctx, key, raw, m.ttl).Err()
}

func HashKey(parts ...string) string {
	h := sha256.New()

	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte("|"))
	}

	return hex.EncodeToString(h.Sum(nil))
}
