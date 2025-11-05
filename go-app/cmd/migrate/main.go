package main

import (
	"log"
	"os"

	"github.com/vitaliisemenov/alert-history/internal/infrastructure/migrations"
)

func main() {
	// Загружаем конфигурацию
	migrationConfig, err := migrations.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load migration config: %v", err)
	}

	backupConfig, err := migrations.LoadBackupConfig()
	if err != nil {
		log.Fatalf("Failed to load backup config: %v", err)
	}

	healthConfig, err := migrations.LoadHealthConfig()
	if err != nil {
		log.Fatalf("Failed to load health config: %v", err)
	}

	// Создаем менеджеры
	manager, err := migrations.NewMigrationManager(migrationConfig)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}

	backupManager := migrations.NewBackupManager(backupConfig, nil, migrationConfig.Logger)

	healthChecker := migrations.NewHealthChecker(nil, healthConfig, migrationConfig.Logger)

	// Создаем CLI
	cli := migrations.NewCLI(manager, backupManager, healthChecker, migrationConfig.Logger)

	// Запускаем CLI
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
