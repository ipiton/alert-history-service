package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Log      LogConfig      `mapstructure:"log"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Lock     LockConfig     `mapstructure:"lock"`
	App      AppConfig      `mapstructure:"app"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port                    int           `mapstructure:"port"`
	Host                    string        `mapstructure:"host"`
	ReadTimeout             time.Duration `mapstructure:"read_timeout"`
	WriteTimeout            time.Duration `mapstructure:"write_timeout"`
	IdleTimeout             time.Duration `mapstructure:"idle_timeout"`
	GracefulShutdownTimeout time.Duration `mapstructure:"graceful_shutdown_timeout"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Database        string        `mapstructure:"database"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConnections  int           `mapstructure:"max_connections"`
	MinConnections  int           `mapstructure:"min_connections"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	QueryTimeout    time.Duration `mapstructure:"query_timeout"`
	URL             string        `mapstructure:"url"`
}

// RedisConfig holds Redis-related configuration
type RedisConfig struct {
	Addr            string        `mapstructure:"addr"`
	Password        string        `mapstructure:"password"`
	DB              int           `mapstructure:"db"`
	PoolSize        int           `mapstructure:"pool_size"`
	MinIdleConns    int           `mapstructure:"min_idle_conns"`
	DialTimeout     time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	MaxRetries      int           `mapstructure:"max_retries"`
	MinRetryBackoff time.Duration `mapstructure:"min_retry_backoff"`
	MaxRetryBackoff time.Duration `mapstructure:"max_retry_backoff"`
}

// LLMConfig holds LLM-related configuration
type LLMConfig struct {
	Provider    string        `mapstructure:"provider"`
	APIKey      string        `mapstructure:"api_key"`
	BaseURL     string        `mapstructure:"base_url"`
	Model       string        `mapstructure:"model"`
	MaxTokens   int           `mapstructure:"max_tokens"`
	Temperature float64       `mapstructure:"temperature"`
	Timeout     time.Duration `mapstructure:"timeout"`
	MaxRetries  int           `mapstructure:"max_retries"`
}

// LogConfig holds logging-related configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// CacheConfig holds cache-related configuration
type CacheConfig struct {
	DefaultTTL      time.Duration `mapstructure:"default_ttl"`
	MaxTTL          time.Duration `mapstructure:"max_ttl"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
	MaxKeys         int64         `mapstructure:"max_keys"`
	EnableMetrics   bool          `mapstructure:"enable_metrics"`
}

// LockConfig holds distributed lock configuration
type LockConfig struct {
	TTL            time.Duration `mapstructure:"ttl"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryInterval  time.Duration `mapstructure:"retry_interval"`
	AcquireTimeout time.Duration `mapstructure:"acquire_timeout"`
	ReleaseTimeout time.Duration `mapstructure:"release_timeout"`
	ValuePrefix    string        `mapstructure:"value_prefix"`
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	Name          string        `mapstructure:"name"`
	Version       string        `mapstructure:"version"`
	Environment   string        `mapstructure:"environment"`
	Debug         bool          `mapstructure:"debug"`
	Timezone      string        `mapstructure:"timezone"`
	MaxWorkers    int           `mapstructure:"max_workers"`
	WorkerTimeout time.Duration `mapstructure:"worker_timeout"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Set default values first
	setDefaults()

	// Enable automatic environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to read configuration file if it exists
	if configPath != "" {
		viper.SetConfigFile(configPath)
		viper.SetConfigType("yaml")

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// Config file not found, continue with defaults and env vars
		}
	}

	// Unmarshal configuration
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// LoadConfigFromEnv loads configuration from environment variables only
func LoadConfigFromEnv() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaults()

	// Unmarshal configuration
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")
	viper.SetDefault("server.graceful_shutdown_timeout", "30s")

	// Database defaults
	viper.SetDefault("database.driver", "postgres")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "alerthistory")
	viper.SetDefault("database.username", "dev")
	viper.SetDefault("database.password", "dev")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.min_connections", 5)
	viper.SetDefault("database.max_conn_lifetime", "1h")
	viper.SetDefault("database.max_conn_idle_time", "30m")
	viper.SetDefault("database.connect_timeout", "10s")
	viper.SetDefault("database.query_timeout", "30s")

	// Redis defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")
	viper.SetDefault("redis.max_retries", 3)
	viper.SetDefault("redis.min_retry_backoff", "100ms")
	viper.SetDefault("redis.max_retry_backoff", "500ms")

	// LLM defaults
	viper.SetDefault("llm.provider", "openai")
	viper.SetDefault("llm.api_key", "")
	viper.SetDefault("llm.base_url", "https://api.openai.com/v1")
	viper.SetDefault("llm.model", "gpt-3.5-turbo")
	viper.SetDefault("llm.max_tokens", 1000)
	viper.SetDefault("llm.temperature", 0.7)
	viper.SetDefault("llm.timeout", "30s")
	viper.SetDefault("llm.max_retries", 3)

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.filename", "")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	viper.SetDefault("log.compress", true)

	// Cache defaults
	viper.SetDefault("cache.default_ttl", "1h")
	viper.SetDefault("cache.max_ttl", "24h")
	viper.SetDefault("cache.cleanup_interval", "10m")
	viper.SetDefault("cache.max_keys", 10000)
	viper.SetDefault("cache.enable_metrics", true)

	// Lock defaults
	viper.SetDefault("lock.ttl", "30s")
	viper.SetDefault("lock.max_retries", 3)
	viper.SetDefault("lock.retry_interval", "100ms")
	viper.SetDefault("lock.acquire_timeout", "5s")
	viper.SetDefault("lock.release_timeout", "2s")
	viper.SetDefault("lock.value_prefix", "lock")

	// App defaults
	viper.SetDefault("app.name", "alert-history")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.environment", "development")
	viper.SetDefault("app.debug", false)
	viper.SetDefault("app.timezone", "UTC")
	viper.SetDefault("app.max_workers", 10)
	viper.SetDefault("app.worker_timeout", "5m")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	if c.Database.Driver == "" {
		return fmt.Errorf("database driver cannot be empty")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if c.Redis.Addr == "" {
		return fmt.Errorf("redis address cannot be empty")
	}

	if c.Log.Level == "" {
		return fmt.Errorf("log level cannot be empty")
	}

	if c.App.Name == "" {
		return fmt.Errorf("app name cannot be empty")
	}

	return nil
}

// GetDatabaseURL constructs database URL from configuration
func (c *Config) GetDatabaseURL() string {
	if c.Database.URL != "" {
		return c.Database.URL
	}

	sslMode := c.Database.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	return fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.Driver,
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		sslMode,
	)
}

// IsDevelopment returns true if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsProduction returns true if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDebug returns true if debug mode is enabled
func (c *Config) IsDebug() bool {
	return c.App.Debug || c.IsDevelopment()
}
