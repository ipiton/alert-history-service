package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// L2Cache is a Redis-backed distributed cache
type L2Cache struct {
	client      *redis.Client
	ttl         time.Duration
	compression bool
	logger      *slog.Logger
}

// NewL2Cache creates a new L2 cache (Redis)
func NewL2Cache(
	addr string,
	password string,
	db int,
	poolSize int,
	minIdle int,
	ttl time.Duration,
	compression bool,
	logger *slog.Logger,
) (*L2Cache, error) {
	if logger == nil {
		logger = slog.Default()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdle,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   3,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("L2 cache (Redis) initialized",
		"addr", addr,
		"db", db,
		"ttl", ttl,
		"compression", compression)

	return &L2Cache{
		client:      client,
		ttl:         ttl,
		compression: compression,
		logger:      logger,
	}, nil
}

// Get retrieves a value from Redis cache
func (c *L2Cache) Get(ctx context.Context, key string) (*core.HistoryResponse, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		c.logger.Error("L2 cache get error", "error", err, "key", key)
		return nil, ErrConnectionFailed
	}

	// Decompress if needed
	if c.compression {
		data, err = c.decompress(data)
		if err != nil {
			c.logger.Error("Failed to decompress L2 cache data", "error", err, "key", key)
			return nil, ErrSerialization("decompression failed", err)
		}
	}

	// Deserialize
	var response core.HistoryResponse
	if err := json.Unmarshal(data, &response); err != nil {
		c.logger.Error("Failed to unmarshal L2 cache data", "error", err, "key", key)
		return nil, ErrSerialization("unmarshal failed", err)
	}

	return &response, nil
}

// Set stores a value in Redis cache
func (c *L2Cache) Set(ctx context.Context, key string, value *core.HistoryResponse) error {
	// Serialize
	data, err := json.Marshal(value)
	if err != nil {
		c.logger.Error("Failed to marshal cache value", "error", err, "key", key)
		return ErrSerialization("marshal failed", err)
	}

	// Compress if enabled
	if c.compression {
		data, err = c.compress(data)
		if err != nil {
			c.logger.Error("Failed to compress cache value", "error", err, "key", key)
			return ErrSerialization("compression failed", err)
		}
	}

	// Store in Redis
	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		c.logger.Error("Failed to set L2 cache", "error", err, "key", key)
		return ErrConnectionFailed
	}

	return nil
}

// Delete removes a key from Redis cache
func (c *L2Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil && err != redis.Nil {
		c.logger.Error("Failed to delete L2 cache key", "error", err, "key", key)
		return ErrConnectionFailed
	}
	return nil
}

// DeletePattern removes all keys matching a pattern
func (c *L2Cache) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var deletedCount int

	for {
		keys, newCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			c.logger.Error("Failed to scan keys", "error", err, "pattern", pattern)
			return ErrConnectionFailed
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				c.logger.Error("Failed to delete keys", "error", err, "pattern", pattern)
				return ErrConnectionFailed
			}
			deletedCount += len(keys)
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	c.logger.Info("Invalidated cache pattern", "pattern", pattern, "deleted_count", deletedCount)
	return nil
}

// compress compresses data using gzip
func (c *L2Cache) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	if _, err := gzipWriter.Write(data); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompress decompresses data using gzip
func (c *L2Cache) decompress(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	return io.ReadAll(gzipReader)
}

// Close closes the Redis connection
func (c *L2Cache) Close() error {
	return c.client.Close()
}

// Ping checks Redis connectivity
func (c *L2Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
