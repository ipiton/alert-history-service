package cache

import (
	"time"
)

// Config contains cache configuration
type Config struct {
	// L1 Cache (in-memory) configuration
	L1Enabled    bool
	L1MaxEntries int64         // Maximum number of entries (default: 10,000)
	L1MaxSizeMB  int64         // Maximum size in MB (default: 100)
	L1TTL        time.Duration // Time-to-live (default: 5 minutes)
	
	// L2 Cache (Redis) configuration
	L2Enabled    bool
	L2TTL        time.Duration // Time-to-live (default: 1 hour)
	L2Compression bool         // Enable gzip compression (default: true)
	
	// Redis connection settings
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisPoolSize int
	RedisMinIdle  int
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *Config {
	return &Config{
		L1Enabled:     true,
		L1MaxEntries:  10000,
		L1MaxSizeMB:   100,
		L1TTL:         5 * time.Minute,
		L2Enabled:     true,
		L2TTL:         1 * time.Hour,
		L2Compression: true,
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       0,
		RedisPoolSize: 50,
		RedisMinIdle:  10,
	}
}

// Validate validates the cache configuration
func (c *Config) Validate() error {
	if c.L1MaxEntries <= 0 {
		return ErrInvalidConfig("L1MaxEntries must be > 0")
	}
	if c.L1MaxSizeMB <= 0 {
		return ErrInvalidConfig("L1MaxSizeMB must be > 0")
	}
	if c.L1TTL <= 0 {
		return ErrInvalidConfig("L1TTL must be > 0")
	}
	if c.L2Enabled && c.L2TTL <= 0 {
		return ErrInvalidConfig("L2TTL must be > 0 when L2 is enabled")
	}
	return nil
}

