package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Alert represents an alert structure
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

// CreateAlertRequest represents alert creation request
type CreateAlertRequest struct {
	Title       string            `json:"title" binding:"required"`
	Description string            `json:"description"`
	Severity    string            `json:"severity" binding:"required"`
	Labels      map[string]string `json:"labels"`
}

var alerts = make([]Alert, 0)
var nextID = 1

func main() {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Routes
	router.GET("/health", healthHandler)
	router.GET("/api/alerts", getAlertsHandler)
	router.GET("/api/alerts/:id", getAlertHandler)
	router.POST("/api/alerts", createAlertHandler)
	router.PUT("/api/alerts/:id", updateAlertHandler)
	router.DELETE("/api/alerts/:id", deleteAlertHandler)

	// Initialize with sample data
	initializeSampleData()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Different port from Fiber
	}

	fmt.Printf("🚀 Gin server starting on port %s\n", port)
	log.Fatal(router.Run(":" + port))
}

// CORS middleware for Gin
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "alert-history-gin",
		"framework": "Gin",
		"timestamp": time.Now().UTC(),
	})
}

func getAlertsHandler(c *gin.Context) {
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start >= len(alerts) {
		c.JSON(http.StatusOK, gin.H{
			"alerts": []Alert{},
			"total":  len(alerts),
		})
		return
	}
	if end > len(alerts) {
		end = len(alerts)
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts[start:end],
		"total":  len(alerts),
	})
}

func getAlertHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid alert ID",
		})
		return
	}

	for _, alert := range alerts {
		if alert.ID == id {
			c.JSON(http.StatusOK, alert)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "Alert not found",
	})
}

func createAlertHandler(c *gin.Context) {
	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON or missing required fields",
		})
		return
	}

	if req.Severity == "" {
		req.Severity = "info"
	}

	alert := Alert{
		ID:          nextID,
		Title:       req.Title,
		Description: req.Description,
		Severity:    req.Severity,
		Status:      "active",
		Labels:      req.Labels,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	alerts = append(alerts, alert)
	nextID++

	c.JSON(http.StatusCreated, alert)
}

func updateAlertHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid alert ID",
		})
		return
	}

	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	for i, alert := range alerts {
		if alert.ID == id {
			alerts[i].Title = req.Title
			alerts[i].Description = req.Description
			alerts[i].Severity = req.Severity
			alerts[i].Labels = req.Labels
			alerts[i].UpdatedAt = time.Now()

			c.JSON(http.StatusOK, alerts[i])
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "Alert not found",
	})
}

func deleteAlertHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid alert ID",
		})
		return
	}

	for i, alert := range alerts {
		if alert.ID == id {
			alerts = append(alerts[:i], alerts[i+1:]...)
			c.JSON(http.StatusOK, gin.H{
				"message": "Alert deleted successfully",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "Alert not found",
	})
}

func initializeSampleData() {
	sampleAlerts := []Alert{
		{
			ID:          1,
			Title:       "Sample Alert 1",
			Description: "This is a sample alert for testing",
			Severity:    "warning",
			Status:      "active",
			Labels:      map[string]string{"service": "web", "env": "prod"},
			CreatedAt:   time.Now().Add(-time.Hour),
			UpdatedAt:   time.Now().Add(-time.Hour),
		},
		{
			ID:          2,
			Title:       "Sample Alert 2",
			Description: "Another sample alert",
			Severity:    "error",
			Status:      "active",
			Labels:      map[string]string{"service": "api", "env": "prod"},
			CreatedAt:   time.Now().Add(-30 * time.Minute),
			UpdatedAt:   time.Now().Add(-30 * time.Minute),
		},
	}

	alerts = append(alerts, sampleAlerts...)
	nextID = 3
}
