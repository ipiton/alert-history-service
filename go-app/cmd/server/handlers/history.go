// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// AlertHistoryItem represents a single alert in the history.
type AlertHistoryItem struct {
	ID          string            `json:"id"`
	AlertName   string            `json:"alertname"`
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
	CreatedAt   string            `json:"createdAt"`
}

// HistoryResponse represents the history API response.
type HistoryResponse struct {
	Alerts     []AlertHistoryItem `json:"alerts"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	Timestamp  string             `json:"timestamp"`
}

// HistoryHandler handles requests to get alert history.
func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Log the history request
	slog.Info("History request received",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"query", r.URL.RawQuery,
	)

	// Only accept GET requests
	if r.Method != http.MethodGet {
		slog.Warn("Invalid HTTP method for history", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Parse page parameter (default: 1)
	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Parse page_size parameter (default: 50, max: 1000)
	pageSize := 50
	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
			if pageSize > 1000 {
				pageSize = 1000
			}
		}
	}

	// Parse status filter
	statusFilter := query.Get("status")

	// Parse alertname filter
	alertNameFilter := query.Get("alertname")

	slog.Debug("History query parameters",
		"page", page,
		"page_size", pageSize,
		"status_filter", statusFilter,
		"alertname_filter", alertNameFilter,
	)

	// Generate mock history data
	alerts, total := generateMockHistory(page, pageSize, statusFilter, alertNameFilter)

	// Create response
	response := HistoryResponse{
		Alerts:    alerts,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode history response", "error", err)
		return
	}

	processingTime := time.Since(startTime)
	slog.Info("History request completed",
		"page", page,
		"page_size", pageSize,
		"total_alerts", total,
		"returned_alerts", len(alerts),
		"processing_time", processingTime,
	)
}

// generateMockHistory generates mock alert history data for testing.
func generateMockHistory(page, pageSize int, statusFilter, alertNameFilter string) ([]AlertHistoryItem, int) {
	// Mock alert templates
	alertTemplates := []AlertHistoryItem{
		{
			AlertName: "HighCPUUsage",
			Status:    "firing",
			Labels: map[string]string{
				"severity": "warning",
				"job":      "node-exporter",
			},
			Annotations: map[string]string{
				"summary":     "High CPU usage detected",
				"description": "CPU usage is above 80%",
			},
		},
		{
			AlertName: "HighMemoryUsage",
			Status:    "firing",
			Labels: map[string]string{
				"severity": "critical",
				"job":      "node-exporter",
			},
			Annotations: map[string]string{
				"summary":     "High memory usage detected",
				"description": "Memory usage is above 90%",
			},
		},
		{
			AlertName: "DiskSpaceLow",
			Status:    "resolved",
			Labels: map[string]string{
				"severity":   "warning",
				"job":        "node-exporter",
				"mountpoint": "/var",
			},
			Annotations: map[string]string{
				"summary":     "Disk space is low",
				"description": "Available disk space is below 10%",
			},
		},
		{
			AlertName: "ServiceDown",
			Status:    "firing",
			Labels: map[string]string{
				"severity": "critical",
				"job":      "blackbox-exporter",
				"service":  "api-gateway",
			},
			Annotations: map[string]string{
				"summary":     "Service is down",
				"description": "HTTP probe failed",
			},
		},
		{
			AlertName: "DatabaseConnectionFailed",
			Status:    "resolved",
			Labels: map[string]string{
				"severity": "critical",
				"job":      "postgres-exporter",
				"database": "production",
			},
			Annotations: map[string]string{
				"summary":     "Database connection failed",
				"description": "Unable to connect to PostgreSQL",
			},
		},
	}

	// Generate mock data
	var allAlerts []AlertHistoryItem
	totalMockAlerts := 10000 // Simulate large dataset

	// Calculate which alerts to return for this page
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize

	alertCount := 0
	for i := 0; i < totalMockAlerts && alertCount < pageSize; i++ {
		// Skip alerts before the start index
		if i < startIndex {
			continue
		}

		// Use template and modify for uniqueness
		template := alertTemplates[i%len(alertTemplates)]
		alert := AlertHistoryItem{
			ID:          strconv.Itoa(totalMockAlerts - i), // Reverse order (newest first)
			AlertName:   template.AlertName,
			Status:      template.Status,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			StartsAt:    time.Now().Add(-time.Duration(i) * time.Minute).UTC().Format(time.RFC3339),
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Minute).UTC().Format(time.RFC3339),
		}

		// Copy labels and annotations
		for k, v := range template.Labels {
			alert.Labels[k] = v
		}
		for k, v := range template.Annotations {
			alert.Annotations[k] = v
		}

		// Add unique instance label
		alert.Labels["instance"] = "server-" + strconv.Itoa((i%100)+1)

		// Set end time for resolved alerts
		if alert.Status == "resolved" {
			alert.EndsAt = time.Now().Add(-time.Duration(i/2) * time.Minute).UTC().Format(time.RFC3339)
		}

		// Apply filters
		if statusFilter != "" && alert.Status != statusFilter {
			continue
		}
		if alertNameFilter != "" && alert.AlertName != alertNameFilter {
			continue
		}

		allAlerts = append(allAlerts, alert)
		alertCount++

		// Stop if we've reached the end index
		if i >= endIndex-1 {
			break
		}
	}

	// Calculate total count considering filters
	total := totalMockAlerts
	if statusFilter != "" || alertNameFilter != "" {
		// In a real implementation, this would be calculated from the database
		// For mock data, we'll estimate based on filter probability
		if statusFilter != "" {
			total = total / 3 // Assume 1/3 of alerts match any given status
		}
		if alertNameFilter != "" {
			total = total / len(alertTemplates) // Divide by number of alert types
		}
	}

	return allAlerts, total
}
