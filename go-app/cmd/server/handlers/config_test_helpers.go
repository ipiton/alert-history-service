package handlers

import (
	"context"
	"time"

	appconfig "github.com/vitaliisemenov/alert-history/internal/config"
)

// mockConfigService is a mock implementation of ConfigService for testing
type mockConfigService struct {
	config *appconfig.Config
	source appconfig.ConfigSource
}

func (m *mockConfigService) GetConfig(ctx context.Context, opts appconfig.GetConfigOptions) (*appconfig.ConfigResponse, error) {
	configMap := map[string]interface{}{
		"Server": map[string]interface{}{
			"port": 8080,
			"host": "localhost",
		},
		"App": map[string]interface{}{
			"name": "test-app",
		},
	}

	return &appconfig.ConfigResponse{
		Version:  "test-version-123",
		Source:   m.source,
		LoadedAt: time.Now(),
		Config:   configMap,
	}, nil
}

func (m *mockConfigService) GetConfigVersion() string {
	return "test-version-123"
}

func (m *mockConfigService) GetConfigSource() appconfig.ConfigSource {
	return m.source
}
