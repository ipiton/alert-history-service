package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAlert представляет тестовую структуру алерта
type TestAlert struct {
	Fingerprint string          `json:"fingerprint"`
	Data        json.RawMessage `json:"data"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func TestSQLiteDatabase_Connect(t *testing.T) {
	// Создаем временный файл для тестирования
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := &Config{
		Driver:         "sqlite",
		SQLiteFile:     dbPath,
		MaxOpenConns:   10,
		MaxIdleConns:   5,
		ConnMaxLifetime: time.Hour,
		Logger:         slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	db, err := NewSQLiteDatabase(config)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Проверяем, что соединение не установлено до вызова Connect
	assert.False(t, db.IsConnected())

	// Устанавливаем соединение
	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)

	// Проверяем, что соединение установлено
	assert.True(t, db.IsConnected())

	// Проверяем, что файл базы данных создан
	_, err = os.Stat(dbPath)
	assert.False(t, os.IsNotExist(err), "Database file should be created")

	// Закрываем соединение
	err = db.Disconnect(ctx)
	require.NoError(t, err)
	assert.False(t, db.IsConnected())
}

func TestSQLiteDatabase_InMemory(t *testing.T) {
	config := &Config{
		Driver: "sqlite",
		SQLiteFile: ":memory:",
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	db, err := NewSQLiteDatabase(config)
	require.NoError(t, err)

	ctx := context.Background()

	err = db.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, db.IsConnected())

	err = db.Disconnect(ctx)
	require.NoError(t, err)
	assert.False(t, db.IsConnected())
}

func TestSQLiteDatabase_Migrate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_migrate.db")

	config := &Config{
		Driver:     "sqlite",
		SQLiteFile: dbPath,
		Logger:     slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	db, err := NewSQLiteDatabase(config)
	require.NoError(t, err)

	ctx := context.Background()

	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Disconnect(ctx)

	// Выполняем миграцию
	err = db.Migrate(ctx)
	require.NoError(t, err)

	// Проверяем, что таблица создана
	row := db.QueryRow(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='alerts'")
	var tableName string
	err = row.Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "alerts", tableName)

	// Проверяем, что индексы созданы
	rows, err := db.Query(ctx, "SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='alerts'")
	require.NoError(t, err)
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes = append(indexes, name)
	}

	assert.Contains(t, indexes, "idx_alerts_created_at")
	assert.Contains(t, indexes, "idx_alerts_updated_at")
}

func TestSQLiteDatabase_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_crud.db")

	config := &Config{
		Driver:     "sqlite",
		SQLiteFile: dbPath,
		Logger:     slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	db, err := NewSQLiteDatabase(config)
	require.NoError(t, err)

	ctx := context.Background()

	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Disconnect(ctx)

	err = db.Migrate(ctx)
	require.NoError(t, err)

	// Создаем тестовый алерт
	alert := TestAlert{
		Fingerprint: "test-fingerprint-123",
		Data:        json.RawMessage(`{"alertname": "TestAlert", "severity": "warning"}`),
	}

	// Вставляем алерт
	result, err := db.Exec(ctx,
		"INSERT INTO alerts (fingerprint, alert_data) VALUES (?, ?)",
		alert.Fingerprint, string(alert.Data))
	require.NoError(t, err)

	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Читаем алерт
	row := db.QueryRow(ctx,
		"SELECT fingerprint, alert_data, created_at, updated_at FROM alerts WHERE fingerprint = ?",
		alert.Fingerprint)

	var retrievedAlert TestAlert
	var dataStr string
	err = row.Scan(&retrievedAlert.Fingerprint, &dataStr, &retrievedAlert.CreatedAt, &retrievedAlert.UpdatedAt)
	require.NoError(t, err)

	retrievedAlert.Data = json.RawMessage(dataStr)
	assert.Equal(t, alert.Fingerprint, retrievedAlert.Fingerprint)
	assert.JSONEq(t, string(alert.Data), string(retrievedAlert.Data))

	// Обновляем алерт
	newData := json.RawMessage(`{"alertname": "UpdatedAlert", "severity": "critical"}`)
	result, err = db.Exec(ctx,
		"UPDATE alerts SET alert_data = ? WHERE fingerprint = ?",
		string(newData), alert.Fingerprint)
	require.NoError(t, err)

	rowsAffected, err = result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Проверяем обновление
	row = db.QueryRow(ctx,
		"SELECT alert_data FROM alerts WHERE fingerprint = ?",
		alert.Fingerprint)

	var updatedDataStr string
	err = row.Scan(&updatedDataStr)
	require.NoError(t, err)
	assert.JSONEq(t, string(newData), updatedDataStr)

	// Удаляем алерт
	result, err = db.Exec(ctx,
		"DELETE FROM alerts WHERE fingerprint = ?",
		alert.Fingerprint)
	require.NoError(t, err)

	rowsAffected, err = result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Проверяем, что алерт удален
	row = db.QueryRow(ctx,
		"SELECT COUNT(*) FROM alerts WHERE fingerprint = ?",
		alert.Fingerprint)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSQLiteDatabase_Transaction(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_transaction.db")

	config := &Config{
		Driver:     "sqlite",
		SQLiteFile: dbPath,
		Logger:     slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	db, err := NewSQLiteDatabase(config)
	require.NoError(t, err)

	ctx := context.Background()

	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Disconnect(ctx)

	err = db.Migrate(ctx)
	require.NoError(t, err)

	// Начинаем транзакцию
	tx, err := db.Begin(ctx)
	require.NoError(t, err)

	// Выполняем операции в транзакции
	alert1 := TestAlert{
		Fingerprint: "tx-test-1",
		Data:        json.RawMessage(`{"alertname": "TxAlert1"}`),
	}

	alert2 := TestAlert{
		Fingerprint: "tx-test-2",
		Data:        json.RawMessage(`{"alertname": "TxAlert2"}`),
	}

	_, err = tx.Exec("INSERT INTO alerts (fingerprint, alert_data) VALUES (?, ?)",
		alert1.Fingerprint, string(alert1.Data))
	require.NoError(t, err)

	_, err = tx.Exec("INSERT INTO alerts (fingerprint, alert_data) VALUES (?, ?)",
		alert2.Fingerprint, string(alert2.Data))
	require.NoError(t, err)

	// Коммитим транзакцию
	err = tx.Commit()
	require.NoError(t, err)

	// Проверяем, что данные сохранены
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM alerts WHERE fingerprint IN (?, ?)",
		alert1.Fingerprint, alert2.Fingerprint)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestSQLiteDatabase_Health(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_health.db")

	config := &Config{
		Driver:     "sqlite",
		SQLiteFile: dbPath,
		Logger:     slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	db, err := NewSQLiteDatabase(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Health должен вернуть ошибку, если соединение не установлено
	err = db.Health(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Disconnect(ctx)

	// Health должен работать после подключения
	err = db.Health(ctx)
	assert.NoError(t, err)
}

func TestSQLiteDatabase_Query(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_query.db")

	config := &Config{
		Driver:     "sqlite",
		SQLiteFile: dbPath,
		Logger:     slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	db, err := NewSQLiteDatabase(config)
	require.NoError(t, err)

	ctx := context.Background()

	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Disconnect(ctx)

	err = db.Migrate(ctx)
	require.NoError(t, err)

	// Вставляем тестовые данные
	alerts := []TestAlert{
		{
			Fingerprint: "query-test-1",
			Data:        json.RawMessage(`{"alertname": "QueryAlert1"}`),
		},
		{
			Fingerprint: "query-test-2",
			Data:        json.RawMessage(`{"alertname": "QueryAlert2"}`),
		},
	}

	for _, alert := range alerts {
		_, err := db.Exec(ctx,
			"INSERT INTO alerts (fingerprint, alert_data) VALUES (?, ?)",
			alert.Fingerprint, string(alert.Data))
		require.NoError(t, err)
	}

	// Выполняем запрос
	rows, err := db.Query(ctx, "SELECT fingerprint, alert_data FROM alerts ORDER BY fingerprint")
	require.NoError(t, err)
	defer rows.Close()

	var retrievedAlerts []TestAlert
	for rows.Next() {
		var alert TestAlert
		var dataStr string
		err := rows.Scan(&alert.Fingerprint, &dataStr)
		require.NoError(t, err)
		alert.Data = json.RawMessage(dataStr)
		retrievedAlerts = append(retrievedAlerts, alert)
	}

	require.Len(t, retrievedAlerts, 2)
	assert.Equal(t, alerts[0].Fingerprint, retrievedAlerts[0].Fingerprint)
	assert.Equal(t, alerts[1].Fingerprint, retrievedAlerts[1].Fingerprint)
}

// BenchmarkSQLiteDatabase_CRUD бенчмарк для CRUD операций
func BenchmarkSQLiteDatabase_CRUD(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "benchmark.db")

	config := &Config{
		Driver:         "sqlite",
		SQLiteFile:     dbPath,
		MaxOpenConns:   10,
		MaxIdleConns:   5,
		ConnMaxLifetime: time.Hour,
		Logger:         slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	db, err := NewSQLiteDatabase(config)
	require.NoError(b, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(b, err)
	defer db.Disconnect(ctx)

	err = db.Migrate(ctx)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fingerprint := fmt.Sprintf("bench-fingerprint-%d", i)
		data := fmt.Sprintf(`{"alertname": "BenchAlert%d", "severity": "info"}`, i)

		// Insert
		_, err := db.Exec(ctx,
			"INSERT INTO alerts (fingerprint, alert_data) VALUES (?, ?)",
			fingerprint, data)
		require.NoError(b, err)

		// Query
		row := db.QueryRow(ctx,
			"SELECT alert_data FROM alerts WHERE fingerprint = ?",
			fingerprint)
		var retrievedData string
		err = row.Scan(&retrievedData)
		require.NoError(b, err)

		// Update
		newData := fmt.Sprintf(`{"alertname": "UpdatedBenchAlert%d", "severity": "warning"}`, i)
		_, err = db.Exec(ctx,
			"UPDATE alerts SET alert_data = ? WHERE fingerprint = ?",
			newData, fingerprint)
		require.NoError(b, err)

		// Delete
		_, err = db.Exec(ctx,
			"DELETE FROM alerts WHERE fingerprint = ?",
			fingerprint)
		require.NoError(b, err)
	}
}
