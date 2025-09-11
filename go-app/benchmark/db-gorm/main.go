package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Alert represents an alert structure for GORM
type Alert struct {
	ID          uint              `json:"id" gorm:"primaryKey"`
	Title       string            `json:"title" gorm:"size:255;not null"`
	Description string            `json:"description" gorm:"type:text"`
	Severity    string            `json:"severity" gorm:"size:20;not null"`
	Status      string            `json:"status" gorm:"size:20;not null;default:'active'"`
	Labels      map[string]string `json:"labels" gorm:"type:jsonb"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

var db *gorm.DB

func main() {
	var err error

	// Get database connection string
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "host=localhost user=postgres password=password dbname=benchmark_db port=5432 sslmode=disable TimeZone=UTC"
	}

	// Connect to database
	db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to PostgreSQL with GORM")

	// Auto-migrate schema
	if err := db.AutoMigrate(&Alert{}); err != nil {
		log.Fatalf("Failed to migrate schema: %v", err)
	}

	fmt.Println("✅ Database schema migrated")

	// Setup HTTP server for benchmarks
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	// Simple HTTP server for database operations
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/alerts", getAlertsHandler)
	http.HandleFunc("/api/alerts/create", createAlertHandler)
	http.HandleFunc("/api/alerts/bulk", bulkInsertHandler)

	fmt.Printf("🚀 GORM server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"service":   "alert-history-gorm",
		"driver":    "GORM",
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
	var alerts []Alert
	result := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&alerts)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
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

	// Create alert
	result := db.Create(&alert)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
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
	tx := db.Begin()
	if tx.Error != nil {
		http.Error(w, tx.Error.Error(), http.StatusInternalServerError)
		return
	}

	// Set defaults and create alerts
	insertedCount := 0
	for i := range alerts {
		if alerts[i].Severity == "" {
			alerts[i].Severity = "info"
		}
		if alerts[i].Status == "" {
			alerts[i].Status = "active"
		}

		if err := tx.Create(&alerts[i]).Error; err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		insertedCount++
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":        "Bulk insert completed",
		"inserted_count": insertedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
