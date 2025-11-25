package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vitaliisemenov/alert-history/internal/core/domain"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/template"
	"github.com/vitaliisemenov/alert-history/internal/notification/template/defaults"
)

// ================================================================================
// TN-155: Template API - Example Templates Seed Script
// ================================================================================
// Seeds database with example templates from TN-154 default templates.
//
// Usage:
//   go run cmd/seed/seed_templates.go -dsn "postgres://..."
//   go run cmd/seed/seed_templates.go -dsn "postgres://..." -clean
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-25

var (
	dsn   = flag.String("dsn", "", "Database connection string")
	clean = flag.Bool("clean", false, "Clean existing templates before seeding")
)

func main() {
	flag.Parse()

	if *dsn == "" {
		log.Fatal("Error: -dsn flag is required\nUsage: go run cmd/seed/seed_templates.go -dsn 'postgres://...'")
	}

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Connect to database
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Initialize repository
	repo, err := template.NewTemplateRepository(pool, logger)
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}

	// Clean existing templates if requested
	if *clean {
		fmt.Println("🧹 Cleaning existing templates...")
		if err := cleanTemplates(ctx, repo); err != nil {
			log.Fatalf("Failed to clean templates: %v", err)
		}
		fmt.Println("✅ Cleaned")
	}

	// Seed templates
	fmt.Println("🌱 Seeding example templates...")

	registry := defaults.GetDefaultTemplates()
	total := 0

	// Seed Slack templates
	fmt.Println("\n📱 Seeding Slack templates...")
	for _, tmpl := range registry.Slack.GetAll() {
		if err := seedTemplate(ctx, repo, tmpl, "slack"); err != nil {
			log.Printf("⚠️  Warning: Failed to seed Slack template %s: %v", tmpl.Name, err)
		} else {
			fmt.Printf("  ✅ %s\n", tmpl.Name)
			total++
		}
	}

	// Seed PagerDuty templates
	fmt.Println("\n📟 Seeding PagerDuty templates...")
	for _, tmpl := range registry.PagerDuty.GetAll() {
		if err := seedTemplate(ctx, repo, tmpl, "pagerduty"); err != nil {
			log.Printf("⚠️  Warning: Failed to seed PagerDuty template %s: %v", tmpl.Name, err)
		} else {
			fmt.Printf("  ✅ %s\n", tmpl.Name)
			total++
		}
	}

	// Seed Email templates
	fmt.Println("\n📧 Seeding Email templates...")
	for _, tmpl := range registry.Email.GetAll() {
		if err := seedTemplate(ctx, repo, tmpl, "email"); err != nil {
			log.Printf("⚠️  Warning: Failed to seed Email template %s: %v", tmpl.Name, err)
		} else {
			fmt.Printf("  ✅ %s\n", tmpl.Name)
			total++
		}
	}

	// Seed WebHook templates
	fmt.Println("\n🔗 Seeding WebHook templates...")
	for _, tmpl := range registry.WebHook.GetAll() {
		if err := seedTemplate(ctx, repo, tmpl, "webhook"); err != nil {
			log.Printf("⚠️  Warning: Failed to seed WebHook template %s: %v", tmpl.Name, err)
		} else {
			fmt.Printf("  ✅ %s\n", tmpl.Name)
			total++
		}
	}

	fmt.Printf("\n🎉 Successfully seeded %d templates!\n", total)
	fmt.Println("\n📊 Template Summary:")
	fmt.Printf("  - Slack: %d templates\n", len(registry.Slack.GetAll()))
	fmt.Printf("  - PagerDuty: %d templates\n", len(registry.PagerDuty.GetAll()))
	fmt.Printf("  - Email: %d templates\n", len(registry.Email.GetAll()))
	fmt.Printf("  - WebHook: %d templates\n", len(registry.WebHook.GetAll()))
	fmt.Printf("  - Total: %d templates\n", total)
	fmt.Println("\n✅ Seeding complete!")
}

func seedTemplate(ctx context.Context, repo template.TemplateRepository, tmpl defaults.Template, templateType string) error {
	// Check if template already exists
	exists, err := repo.Exists(ctx, tmpl.Name)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}

	if exists {
		return fmt.Errorf("template already exists (use -clean to remove existing)")
	}

	// Create domain template
	domainTemplate := &domain.Template{
		Name:        tmpl.Name,
		Type:        domain.TemplateType(templateType),
		Content:     tmpl.Template,
		Description: tmpl.Description,
		Metadata: map[string]interface{}{
			"source":   "TN-154 default templates",
			"category": tmpl.Name,
			"author":   "system",
			"tags":     []string{templateType, "default", "example"},
		},
		CreatedBy: "seed-script",
		Version:   1,
	}

	// Create template
	if err := repo.Create(ctx, domainTemplate); err != nil {
		return fmt.Errorf("failed to create: %w", err)
	}

	return nil
}

func cleanTemplates(ctx context.Context, repo template.TemplateRepository) error {
	// Get all templates
	filters := domain.ListFilters{
		Limit:  1000,
		Offset: 0,
	}

	templates, _, err := repo.List(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	// Delete each template (hard delete)
	for _, tmpl := range templates {
		if err := repo.Delete(ctx, tmpl.Name, true); err != nil {
			log.Printf("⚠️  Warning: Failed to delete template %s: %v", tmpl.Name, err)
		}
	}

	fmt.Printf("  Deleted %d templates\n", len(templates))
	return nil
}
