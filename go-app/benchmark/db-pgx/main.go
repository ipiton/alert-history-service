package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Alert represents an alert structure for database operations
type Alert struct {
	ID          int                    `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Status      string                 `json:"status"`
	Labels      map[string]string      `json:"labels"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

var db *pgxpool.Pool

func main() {
	var err error

	// Get database connection string
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/benchmark_db?sslmode=disable"
	}

	// Connect to database
	db, err = pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to PostgreSQL with pgx")

	// Setup database schema
	if err := setupSchema(); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}

	fmt.Println("✅ Database schema ready")

	// Setup HTTP server for benchmarks
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	// Simple HTTP server for database operations
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/alerts", getAlertsHandler)
	http.HandleFunc("/api/alerts/create", createAlertHandler)
	http.HandleFunc("/api/alerts/bulk", bulkInsertHandler)

	fmt.Printf("🚀 pgx server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func setupSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS alerts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			severity VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			labels JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
		CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
		CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);
		CREATE INDEX IF NOT EXISTS idx_alerts_labels ON alerts USING GIN(labels);
	`

	_, err := db.Exec(context.Background(), schema)
	return err
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"service":   "alert-history-pgx",
		"driver":    "pgx",
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getAlertsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "10"
	}

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offsetStr = "0"
	}

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Query alerts
	rows, err := db.Query(context.Background(),
		`SELECT id, title, description, severity, status, labels, created_at, updated_at
		 FROM alerts
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var alert Alert
		err := rows.Scan(&alert.ID, &alert.Title, &alert.Description,
			&alert.Severity, &alert.Status, &alert.Labels,
			&alert.CreatedAt, &alert.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		alerts = append(alerts, alert)
	}

	response := map[string]interface{}{
		"alerts": alerts,
		"total":  len(alerts),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createAlertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var alert Alert
	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set defaults
	if alert.Severity == "" {
		alert.Severity = "info"
	}
	if alert.Status == "" {
		alert.Status = "active"
	}
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()

	// Insert alert
	err := db.QueryRow(context.Background(),
		`INSERT INTO alerts (title, description, severity, status, labels, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		alert.Title, alert.Description, alert.Severity, alert.Status,
		alert.Labels, alert.CreatedAt, alert.UpdatedAt).Scan(&alert.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alert)
}

func bulkInsertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var alerts []Alert
	if err := json.NewDecoder(r.Body).Decode(&alerts); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := db.Begin(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	// Insert all alerts (using simple exec for each insert)
	insertedCount := 0
	for _, alert := range alerts {
		if alert.Severity == "" {
			alert.Severity = "info"
		}
		if alert.Status == "" {
			alert.Status = "active"
		}
		alert.CreatedAt = time.Now()
		alert.UpdatedAt = time.Now()

		_, err := tx.Exec(context.Background(),
			`INSERT INTO alerts (title, description, severity, status, labels, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			alert.Title, alert.Description, alert.Severity, alert.Status,
			alert.Labels, alert.CreatedAt, alert.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		insertedCount++
	}

	// Commit transaction
	if err := tx.Commit(context.Background()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":         "Bulk insert completed",
		"inserted_count": insertedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
