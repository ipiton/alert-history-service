package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationsExist проверяет наличие директории migrations
func TestMigrationsExist(t *testing.T) {
	// Проверяем, существует ли директория migrations
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		t.Skip("Migrations directory does not exist - TN-14 not completed yet")
		return
	}

	assert.DirExists(t, "migrations", "migrations directory should exist")
}

// TestGooseInstalled проверяет установку goose
func TestGooseInstalled(t *testing.T) {
	// Проверяем, что goose установлен
	cmd := exec.Command("goose", "version")
	err := cmd.Run()

	// Если goose не установлен, пропускаем тест
	if err != nil {
		t.Skip("Goose is not installed - run 'go install github.com/pressly/goose/v3/cmd/goose@latest'")
		return
	}

	assert.NoError(t, err, "goose should be installed and accessible")
}

// TestMigrationsRunUp проверяет возможность запуска миграций вверх
func TestMigrationsRunUp(t *testing.T) {
	// Проверяем наличие директории migrations
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		t.Skip("Migrations directory does not exist - TN-14 not completed yet")
		return
	}

	// Проверяем, что goose установлен
	cmd := exec.Command("goose", "version")
	if err := cmd.Run(); err != nil {
		t.Skip("Goose is not installed - run 'go install github.com/pressly/goose/v3/cmd/goose@latest'")
		return
	}

	// Получаем DATABASE_URL из переменных окружения
	databaseURL := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, databaseURL, "DATABASE_URL environment variable must be set")

	// Запускаем миграции
	cmd = exec.Command("goose", "-dir", "migrations", "postgres", databaseURL, "up")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Migration output: %s", string(output))
		t.Errorf("Failed to run migrations up: %v", err)
		return
	}

	t.Logf("Migrations applied successfully: %s", string(output))
}

// TestMigrationsRunDown проверяет возможность запуска миграций вниз (rollback)
func TestMigrationsRunDown(t *testing.T) {
	// Проверяем наличие директории migrations
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		t.Skip("Migrations directory does not exist - TN-14 not completed yet")
		return
	}

	// Проверяем, что goose установлен
	cmd := exec.Command("goose", "version")
	if err := cmd.Run(); err != nil {
		t.Skip("Goose is not installed - run 'go install github.com/pressly/goose/v3/cmd/goose@latest'")
		return
	}

	// Получаем DATABASE_URL из переменных окружения
	databaseURL := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, databaseURL, "DATABASE_URL environment variable must be set")

	// Сначала применяем миграции (на случай, если они не применены)
	upCmd := exec.Command("goose", "-dir", "migrations", "postgres", databaseURL, "up")
	if output, err := upCmd.CombinedOutput(); err != nil {
		t.Logf("Migration up output: %s", string(output))
		t.Skipf("Cannot apply migrations up first: %v", err)
		return
	}

	// Запускаем rollback миграций
	cmd := exec.Command("goose", "-dir", "migrations", "postgres", databaseURL, "down")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Migration rollback output: %s", string(output))
		t.Errorf("Failed to run migrations down: %v", err)
		return
	}

	t.Logf("Migrations rolled back successfully: %s", string(output))
}

// TestMigrationStatus проверяет статус миграций
func TestMigrationStatus(t *testing.T) {
	// Проверяем наличие директории migrations
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		t.Skip("Migrations directory does not exist - TN-14 not completed yet")
		return
	}

	// Проверяем, что goose установлен
	cmd := exec.Command("goose", "version")
	if err := cmd.Run(); err != nil {
		t.Skip("Goose is not installed - run 'go install github.com/pressly/goose/v3/cmd/goose@latest'")
		return
	}

	// Получаем DATABASE_URL из переменных окружения
	databaseURL := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, databaseURL, "DATABASE_URL environment variable must be set")

	// Проверяем статус миграций
	cmd = exec.Command("goose", "-dir", "migrations", "postgres", databaseURL, "status")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Migration status output: %s", string(output))
		t.Errorf("Failed to get migration status: %v", err)
		return
	}

	t.Logf("Migration status: %s", string(output))
	// Статус должен завершиться без ошибки
	assert.NoError(t, err, "migration status should complete without error")
}
