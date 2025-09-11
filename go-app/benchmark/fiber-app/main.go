package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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
	Title       string            `json:"title" validate:"required"`
	Description string            `json:"description"`
	Severity    string            `json:"severity" validate:"required"`
	Labels      map[string]string `json:"labels"`
}

var alerts = make([]Alert, 0)
var nextID = 1

func main() {
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())

	// Routes
	app.Get("/health", healthHandler)
	app.Get("/api/alerts", getAlertsHandler)
	app.Get("/api/alerts/:id", getAlertHandler)
	app.Post("/api/alerts", createAlertHandler)
	app.Put("/api/alerts/:id", updateAlertHandler)
	app.Delete("/api/alerts/:id", deleteAlertHandler)

	// Initialize with sample data
	initializeSampleData()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Fiber server starting on port %s\n", port)
	log.Fatal(app.Listen(":" + port))
}

func healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"service":   "alert-history-fiber",
		"framework": "Fiber v2",
		"timestamp": time.Now().UTC(),
	})
}

func getAlertsHandler(c *fiber.Ctx) error {
	// Parse query parameters
	limitStr := c.Query("limit", "10")
	offsetStr := c.Query("offset", "0")

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
		return c.JSON(fiber.Map{
			"alerts": []Alert{},
			"total":  len(alerts),
		})
	}
	if end > len(alerts) {
		end = len(alerts)
	}

	return c.JSON(fiber.Map{
		"alerts": alerts[start:end],
		"total":  len(alerts),
	})
}

func getAlertHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	for _, alert := range alerts {
		if alert.ID == id {
			return c.JSON(alert)
		}
	}

	return c.Status(404).JSON(fiber.Map{
		"error": "Alert not found",
	})
}

func createAlertHandler(c *fiber.Ctx) error {
	var req CreateAlertRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid JSON",
		})
	}

	// Basic validation
	if req.Title == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Title is required",
		})
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

	return c.Status(201).JSON(alert)
}

func updateAlertHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	var req CreateAlertRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid JSON",
		})
	}

	for i, alert := range alerts {
		if alert.ID == id {
			alerts[i].Title = req.Title
			alerts[i].Description = req.Description
			alerts[i].Severity = req.Severity
			alerts[i].Labels = req.Labels
			alerts[i].UpdatedAt = time.Now()

			return c.JSON(alerts[i])
		}
	}

	return c.Status(404).JSON(fiber.Map{
		"error": "Alert not found",
	})
}

func deleteAlertHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	for i, alert := range alerts {
		if alert.ID == id {
			alerts = append(alerts[:i], alerts[i+1:]...)
			return c.JSON(fiber.Map{
				"message": "Alert deleted successfully",
			})
		}
	}

	return c.Status(404).JSON(fiber.Map{
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
