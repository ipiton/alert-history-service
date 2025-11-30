//go:build integration || e2e
// +build integration e2e

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestInfrastructure manages test infrastructure lifecycle
type TestInfrastructure struct {
	PostgresContainer *postgres.PostgresContainer
	RedisContainer    testcontainers.Container
	DB                *sql.DB
	RedisClient       *redis.Client
	MockLLMServer     *MockLLMServer
	BaseURL           string
	ctx               context.Context
}

// Context returns the infrastructure context
func (ti *TestInfrastructure) Context() context.Context {
	return ti.ctx
}

// SetupTestInfrastructure starts all required infrastructure
func SetupTestInfrastructure(ctx context.Context) (*TestInfrastructure, error) {
	infra := &TestInfrastructure{
		ctx: ctx,
	}

	// Start PostgreSQL container
	if err := infra.startPostgres(ctx); err != nil {
		return nil, fmt.Errorf("failed to start postgres: %w", err)
	}

	// Start Redis container
	if err := infra.startRedis(ctx); err != nil {
		infra.Teardown(ctx) // Cleanup postgres
		return nil, fmt.Errorf("failed to start redis: %w", err)
	}

	// Start Mock LLM Server
	infra.MockLLMServer = NewMockLLMServer()

	// Set base URL (will be configured when app starts)
	infra.BaseURL = "http://localhost:8080/api/v2"

	return infra, nil
}

// startPostgres starts PostgreSQL container with testcontainers
func (ti *TestInfrastructure) startPostgres(ctx context.Context) error {
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("alerthistory_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	ti.PostgresContainer = pgContainer

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	// Connect to database (using pgx driver)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping postgres: %w", err)
	}

	ti.DB = db

	return nil
}

// startRedis starts Redis container with testcontainers
func (ti *TestInfrastructure) startRedis(ctx context.Context) error {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(30 * time.Second),
	}

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start redis container: %w", err)
	}

	ti.RedisContainer = redisContainer

	// Get Redis host and port
	host, err := redisContainer.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get redis host: %w", err)
	}

	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		return fmt.Errorf("failed to get redis port: %w", err)
	}

	// Connect to Redis
	redisAddr := fmt.Sprintf("%s:%s", host, port.Port())
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	// Test connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisClient.Close()
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	ti.RedisClient = redisClient

	return nil
}

// Teardown stops all infrastructure
func (ti *TestInfrastructure) Teardown(ctx context.Context) error {
	var errs []error

	// Close Mock LLM Server
	if ti.MockLLMServer != nil {
		ti.MockLLMServer.Close()
	}

	// Close Redis client
	if ti.RedisClient != nil {
		if err := ti.RedisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close redis client: %w", err))
		}
	}

	// Stop Redis container
	if ti.RedisContainer != nil {
		if err := ti.RedisContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate redis container: %w", err))
		}
	}

	// Close database connection
	if ti.DB != nil {
		if err := ti.DB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close database: %w", err))
		}
	}

	// Stop PostgreSQL container
	if ti.PostgresContainer != nil {
		if err := ti.PostgresContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate postgres container: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("teardown errors: %v", errs)
	}

	return nil
}

// ResetDatabase truncates all tables for clean test state
func (ti *TestInfrastructure) ResetDatabase(ctx context.Context) error {
	// Truncate all tables (in correct order to avoid FK violations)
	tables := []string{
		"alert_history",
		"silences",
		"inhibition_states",
		"classification_cache",
		"alert_groups",
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := ti.DB.ExecContext(ctx, query); err != nil {
			// Ignore errors for non-existent tables (tests may not have all tables)
			continue
		}
	}

	return nil
}

// ResetRedis flushes Redis database
func (ti *TestInfrastructure) ResetRedis(ctx context.Context) error {
	if ti.RedisClient == nil {
		return nil
	}
	return ti.RedisClient.FlushDB(ctx).Err()
}

// GetPostgresConnString returns PostgreSQL connection string
func (ti *TestInfrastructure) GetPostgresConnString(ctx context.Context) (string, error) {
	if ti.PostgresContainer == nil {
		return "", fmt.Errorf("postgres container not started")
	}
	return ti.PostgresContainer.ConnectionString(ctx, "sslmode=disable")
}

// GetRedisAddr returns Redis address
func (ti *TestInfrastructure) GetRedisAddr(ctx context.Context) (string, error) {
	if ti.RedisContainer == nil {
		return "", fmt.Errorf("redis container not started")
	}

	host, err := ti.RedisContainer.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get redis host: %w", err)
	}

	port, err := ti.RedisContainer.MappedPort(ctx, "6379")
	if err != nil {
		return "", fmt.Errorf("failed to get redis port: %w", err)
	}

	return fmt.Sprintf("%s:%s", host, port.Port()), nil
}

// WaitForHealthy waits for all services to be healthy
func (ti *TestInfrastructure) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for services to be healthy")
		case <-ticker.C:
			// Check PostgreSQL
			if err := ti.DB.PingContext(ctx); err != nil {
				continue
			}

			// Check Redis
			if err := ti.RedisClient.Ping(ctx).Err(); err != nil {
				continue
			}

			// All healthy
			return nil
		}
	}
}
